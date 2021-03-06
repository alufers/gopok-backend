package blog

import (
	"net/http"
	"strconv"
	"time"

	"github.com/fatih/structs"
	"github.com/gopok/gopok-backend/pkg/auth"
	"github.com/gopok/gopok-backend/pkg/core"
	"github.com/gorilla/mux"
	"gopkg.in/mgo.v2/bson"
)

/*
PostsController handles actions with posts.
*/
type PostsController struct {
	app             *core.Application
	postsController *mux.Router
}

/*
Register registers the controller
*/
func (pc *PostsController) Register(app *core.Application) {
	pc.app = app
	pc.postsController = app.Router.PathPrefix("/api/blog/posts").Subrouter()
	pc.postsController.Handle("", auth.CheckUserMiddleware(app)(http.HandlerFunc(core.WrapRest(pc.addPost)))).Methods("POST")
	pc.postsController.Handle("/{id}/comments", auth.CheckUserMiddleware(app)(http.HandlerFunc(core.WrapRest(pc.addComment)))).Methods("POST")
	pc.postsController.HandleFunc("/new", core.WrapRest(pc.getNewPosts)).Methods("GET")
	pc.postsController.HandleFunc("/{id}", core.WrapRest(pc.getPostByID)).Methods("GET")
	pc.postsController.Handle("/{id}/upvote", auth.CheckUserMiddleware(app)(http.HandlerFunc(core.WrapRest(pc.upvotePost)))).Methods("POST")
	pc.postsController.Handle("/{id}/downvote", auth.CheckUserMiddleware(app)(http.HandlerFunc(core.WrapRest(pc.downvotePost)))).Methods("POST")
	pc.postsController.Handle("/{id}/comments/{commentID}/upvote", auth.CheckUserMiddleware(app)(http.HandlerFunc(core.WrapRest(pc.upvoteComment)))).Methods("POST")
	pc.postsController.Handle("/{id}/comments/{commentID}/downvote", auth.CheckUserMiddleware(app)(http.HandlerFunc(core.WrapRest(pc.downvoteComment)))).Methods("POST")
}

func (pc *PostsController) addPost(r *core.RestRequest) interface{} {
	user := r.OriginalRequest.Context().Value(auth.UserContextKey).(*auth.User)
	var allData map[string]string
	jsonErr := r.DecodeJSON(&allData)
	if jsonErr != nil {
		return core.NewErrorResponse("invalid JSON request: "+jsonErr.Error(), 400)
	}
	p := &Post{
		Title: 		allData["title"]
		Content:    allData["content"],
		AuthorID:   user.ID,
		Comments:   []Comment{},
		Downvoters: []bson.ObjectId{},
		Upvoters:   []bson.ObjectId{},
	}
	validationError := p.Validate()
	if validationError != nil {
		return validationError
	}
	p.ID = bson.NewObjectId()
	p.CreatedOn = time.Now()
	err := pc.app.Db.C("posts").Insert(&p)
	if err != nil {
		return core.NewErrorResponse(err.Error(), 500)
	}
	return pc.attachAuthorToPost(p)
}

func (pc *PostsController) attachAuthorToPost(p *Post) map[string]interface{} {

	pp := structs.Map(p)
	author := &auth.User{}
	pc.app.Db.C("users").FindId(p.AuthorID).One(author)
	pp["author"] = author

	populatedComments := make([]map[string]interface{}, 0)
	for _, c := range p.Comments {
		populatedComments = append(populatedComments, pc.attachAuthorToComment(&c))
	}
	pp["comments"] = populatedComments
	return pp
}

func (pc *PostsController) getNewPosts(r *core.RestRequest) interface{} {
	posts := []Post{}
	afterStr := r.OriginalRequest.URL.Query().Get("after")

	var after time.Time
	if afterStr == "" {
		after = time.Now()
	} else {
		afterNum, parseErr := strconv.ParseInt(afterStr, 10, 64)
		if parseErr != nil {
			return core.NewErrorResponse("after should be a string convertable to int64", 400)
		}
		after = time.Unix(0, afterNum)
	}
	findAllErr := pc.app.Db.C("posts").Find(bson.M{
		"createdOn": bson.M{
			"$lt": after,
		},
	}).Sort("-createdOn").Limit(20).All(&posts)
	if findAllErr != nil {
		return core.NewErrorResponse(findAllErr.Error(), 500)
	}
	populatedPosts := make([]map[string]interface{}, 0)

	for _, p := range posts {

		populatedPosts = append(populatedPosts, pc.attachAuthorToPost(&p))
	}
	var nextCursor string
	if len(posts) > 0 {

		nextCursor = strconv.FormatInt(posts[len(posts)-1].CreatedOn.UnixNano(), 10)
	} else {
		nextCursor = "0"
	}

	return map[string]interface{}{
		"posts":      populatedPosts,
		"nextCursor": nextCursor,
	}
}

func (pc *PostsController) getPostByID(r *core.RestRequest) interface{} {
	postID := mux.Vars(r.OriginalRequest)["id"]
	post := &Post{}
	if bson.IsObjectIdHex(postID) {

		findErr := pc.app.Db.C("posts").FindId(bson.ObjectIdHex(postID)).One(post)
		if findErr != nil {
			return core.NewErrorResponse(findErr.Error(), 500)
		}
	} else {
		return core.NewErrorResponse("not found", 404)
	}

	return pc.attachAuthorToPost(post)
}

func init() {
	core.ControllersToRegister.PushBack(&PostsController{})
}
