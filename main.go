package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/swagger"
	"github.com/joho/godotenv"

	_ "github.com/2miwon/video-streaming/docs"
	"golang.org/x/crypto/bcrypt"
)

// @Summary Register a new user
// @Description Register a new user with email, username and password
// @Tags users
// @Accept  json
// @Produce  json
// @Param   email     body    string     true        "Email"
// @Param   username  body    string     true        "Username"
// @Param   password  body    string     true        "Password"
// @Success 200 {object} User
// @Failure 400 {object} string "User already exists"
// @Failure 500 {object} string "Internal server error"
// @Router /register [post]
func registerUser(c *fiber.Ctx, ctx context.Context, db *mongo.Database) error {
	collection := db.Collection("users")
	body := jsonParser(c)

	if body["email"] == nil || body["password"] == nil || body["username"] == nil {
		return c.SendStatus(400)
	}

	filter := bson.M{"email": body["email"].(string)}
	
	err := checkDocumentNotExists(collection, ctx, filter, "User already exists")
	if err != nil {
		return c.SendStatus(400)
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(body["password"].(string)), bcrypt.DefaultCost)
	if err != nil {
		return c.SendStatus(500)
	}
	token, err := bcrypt.GenerateFromPassword([]byte(body["email"].(string)), bcrypt.DefaultCost)
	if err != nil {
		return c.SendStatus(500)
	}
	user := User{
		Email: body["email"].(string),
		Username: body["username"].(string),
		Password: string(hashedPassword),
		Created: time.Now(),
		Token: string(token),
		BookMark: []string{},
	}
	rst, err := createUser(collection, ctx, user)
	if err != nil {
		return c.SendStatus(500)
	}
	return c.JSON(rst)
}

// @Summary Get user info
// @Description Get user info with token
// @Tags users
// @Accept  json
// @Produce  json
// @Param   token     body    string     true        "User token"
// @Success 200 {object} User
// @Failure 403 {object} string "User not found"
// @Router /user/my_info [post]
func getMyInfo(c *fiber.Ctx, ctx context.Context, db *mongo.Database) error {
	collection := db.Collection("users")

	body := jsonParser(c)
	var rst bson.M
	if body["token"] == nil {
		return c.SendStatus(403)
	}
	err := collection.FindOne(ctx, bson.M{"token": body["token"].(string)}).Decode(&rst)
	if err != nil {
		return c.SendStatus(403)
	}

	return c.JSON(rst)
}

// @Summary Create a new video
// @Description Create a new video with title, content, url, author_id
// @Tags videos
// @Accept  json
// @Produce  json
// @Param   title     body    string     true        "Title"
// @Param   content   body    string     true        "Content"
// @Param   url       body    string     true        "URL"
// @Param   author_id body    string     true        "Author ID"
// @Success 200 {object} Video
// @Failure 500 {object} string "Internal server error"
// @Router /videos/create [post]
func createVideo(c *fiber.Ctx, ctx context.Context, db *mongo.Database) error {
	collection := db.Collection("videos")
	body := jsonParser(c)

	if body["title"] == nil || body["content"] == nil || body["url"] == nil || body["author_id"] == nil {
		return c.SendStatus(400)
	}

	video := Video{
		Title: body["title"].(string),
		Content: body["content"].(string),
		URL: body["url"].(string),
		AuthorID: body["author_id"].(string),
		Created: time.Now(),
	}

	// if body["thumbnail_url"] != nil {
	// 	video.(bson.M)["thumbnail_url"] = body["thumbnail_url"]
	// }

	rst, err := collection.InsertOne(ctx, video)
	if err != nil {
		return c.SendStatus(500)
	}

	return c.JSON(rst)
}

// @Summary Get all videos
// @Description Get all videos
// @Tags videos
// @Produce  json
// @Success 200 {object} Video
// @Failure 500 {object} string "Internal server error"
// @Router /videos/all [get]
func getAllVideos(c *fiber.Ctx, ctx context.Context, db *mongo.Database) error {
	collection := db.Collection("videos")
	cursor, err := collection.Find(ctx, bson.M{})
	checkErr(err)

	var videos []Video
	if err = cursor.All(ctx, &videos); err != nil {
		return c.SendStatus(500)
	}

	return c.JSON(videos)
}

// @Summary Delete a video
// @Description Delete a video with video_id and author_id
// @Tags videos
// @Accept  json
// @Produce  json
// @Param   video_id     body    string     true        "Video ID"
// @Param   my_id        body    string     true        "My ID"
// @Success 200 {object} Video
// @Failure 400 {object} string "Video not found"
// @Failure 500 {object} string "Internal server error"
// @Router /videos/delete [post]
func deleteVideo(c *fiber.Ctx, ctx context.Context, db *mongo.Database) error {
	collection := db.Collection("videos")
		body := jsonParser(c)

		if body["video_id"] == nil || body["my_id"] == nil {
			return c.SendStatus(400)
		}

		filter := bson.M{
			"_id": body["video_id"],
			"author_id": body["my_id"],
		}

		err := checkDocumentExists(collection, ctx, filter, "Video not found")
		if err != nil {
			return c.SendStatus(400)
		}

		update := bson.M{
			"$set": bson.M{
				"deleted": primitive.Timestamp{T: uint32(time.Now().Unix())},
			},
		}
		rst, err := collection.UpdateOne(ctx, filter, update)
		if err != nil {
			return c.SendStatus(500)
		}
		return c.JSON(rst)
}

// @Summary Get my videos
// @Description Get my videos with author_id
// @Tags videos
// @Produce  json
// @Param   id     path    string     true        "Author ID"
// @Success 200 {object} Video
// @Failure 400 {object} string "Internal server error"
// @Router /videos/user/{id} [get]

func getMyVideos(c *fiber.Ctx, ctx context.Context, db *mongo.Database) error {
	collection := db.Collection("videos")

	id := c.Params("id")

	if id == "" {
		return c.SendStatus(400)
	}
	

	cursor, err := collection.Find(ctx, bson.M{"author_id": id})
	checkErr(err)

	var videos []Video
	if err = cursor.All(ctx, &videos); err != nil {
		return c.SendStatus(500)
	}

	return c.JSON(videos)
}

// @Summary Get video info
// @Description Get video info with video_id
// @Tags videos
// @Produce  json
// @Param   video_id     path    string     true        "Video ID"
// @Success 200 {object} Video
// @Failure 400 {object} string "Internal server error"
// @Router /videos/info/{video_id} [get]
func getVideoInfo(c *fiber.Ctx, ctx context.Context, db *mongo.Database) error {
	collection := db.Collection("videos")
	rst, err := collection.Find(ctx, bson.M{"_id": c.Params("video_id")})
	if err != nil {
		return c.SendStatus(400)
	}

	return c.JSON(rst)
}

// @Summary Login
// @Description Login with email and password
// @Tags users
// @Accept  json
// @Produce  json
// @Param   email     body    string     true        "Email"
// @Param   password  body    string     true        "Password"
// @Success 200 {object} string
// @Failure 400 {object} string "User not found"
// @Failure 403 {object} string "Invalid password"
// @Failure 500 {object} string "Internal server error"
// @Router /login [post]
func login(c *fiber.Ctx, ctx context.Context, db *mongo.Database) error {
	collection := db.Collection("users")
		body := jsonParser(c)

		if body["email"] == nil || body["password"] == nil {
			return c.SendStatus(400)
		}

		filter := bson.M{"email": body["email"].(string)}
		err := checkDocumentExists(collection, ctx, filter, "User not found")
		if err != nil {
			return c.SendStatus(400)
		}

		user := User{}
		
		err = collection.FindOne(ctx, filter).Decode(&user)
		if err != nil {
			return c.SendStatus(500)
		}

		err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(body["password"].(string)))
		if err != nil {
			return c.SendStatus(403)
		}
		
		return c.JSON(user.Token)
}

// @Summary Update user
// @Description Update user with token
// @Tags users
// @Accept  json
// @Produce  json
// @Param   token     body    string     true        "User token"
// @Param   video_history  body    string     false        "Video history"
// @Param   add_bookmark   body    string     false        "Add bookmark"
// @Param   delete_bookmark   body    string     false        "Delete bookmark"
// @Success 200 {object} string
// @Failure 400 {object} string "User not found"
// @Failure 500 {object} string "Internal server error"
// @Router /user/update [post]
func updateUser(c *fiber.Ctx, ctx context.Context, db *mongo.Database) error {
	collection := db.Collection("users")
	body := jsonParser(c)

	if body["token"] == nil {
		return c.SendStatus(400)
	}

	filter := bson.M{"token": body["token"]}
	err := checkDocumentExists(collection, ctx, filter, "User not found")
	if err != nil {
		return c.SendStatus(400)
	}

	add := bson.M{
		"$push": bson.M{},
	}

	del := bson.M{
		"$pull": bson.M{},
	}

	if body["video_history"] != nil {
		rst := User{}
		err := collection.FindOne(ctx, bson.M{"history": bson.M{"VideoID": body["video_history"]}}).Decode(&rst)
		if err != nil {
			remove := bson.M{"$pull": bson.M{"history": bson.M{"VideoID": body["video_history"]}}}
    		//_, _ := 
			collection.UpdateOne(context.TODO(), filter, remove)
    		// if err != nil {
    		// } 
		}

		videoHistory := VideoHistory{
			VideoID: body["video_history"].(string),
			Date:    time.Now(),
		}

		add := bson.M{"$push": bson.M{"history": videoHistory}}
   		_, err = collection.UpdateOne(context.TODO(), filter, add)
   		if err != nil {
   		    log.Fatal(err)
   		}
	}

	if body["add_bookmark"] != nil {
		add["$push"] = bson.M{"bookmark": body["add_bookmark"].(string)}
	}

	if body["delete_bookmark"] != nil {
		del["$pull"] = bson.M{"bookmark": body["delete_bookmark"].(string)}
	}

	// if body["username"] != nil {
	// 	update["$set"].(bson.M)["username"] = body["username"]
	// }

	// if body["password"] != nil {
	// 	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(body["password"].(string)), bcrypt.DefaultCost)
	// 	if err != nil {
	// 		return c.SendStatus(500)
	// 	}
	// 	update["$set"].(bson.M)["password"] = string(hashedPassword)
	// }

	_, err = collection.UpdateOne(ctx, filter, add)
	if err != nil {
	    return c.SendStatus(500)
	}
	
	_, err = collection.UpdateOne(ctx, filter, del)
	if err != nil {
	    return c.SendStatus(500)
	}

	return c.SendStatus(200)
}

// @Summary Add comment
// @Description Add comment to video
// @Tags videos
// @Accept  json
// @Produce  json
// @Param   video_id     body    string     true        "Video ID"
// @Param   content  body    string     true        "Add comment"
// @Success 200 {object} string
// @Failure 400 {object} string "Video not found"
// @Failure 500 {object} string "Internal server error"
// @Router /video/comment [post]
func addComment(c *fiber.Ctx, ctx context.Context, db *mongo.Database) error {
	collection := db.Collection("videos")
	body := jsonParser(c)

	if body["video_id"] == nil {
		return c.SendStatus(400)
	}

	filter := bson.M{
		"_id": body["video_id"],
	}
	err := checkDocumentExists(collection, ctx, filter, "Video not found")
	if err != nil {
		return c.SendStatus(400)
	}

	add := bson.M{
		"$push": bson.M{},
	}

	if body["comment"] != nil {
		add["$push"] = bson.M{"comments": body["content"].(string)}
	}

	_, err = collection.UpdateOne(ctx, filter, add)
	if err != nil {
	    return c.SendStatus(500)
	}
	
	return c.SendStatus(200)
}

// @title SuperNova API
// @version 1.0
// @description This is a swagger docs for Fiber
// @termsOfService http://swagger.io/terms/
// @contact.name API Support
// @contact.email fiber@swagger.io
// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html
// @host 3.36.212.250:3000
// @BasePath /docs
func main() {
	err := godotenv.Load()
	checkErr(err)
	db_uri := os.Getenv("DB_URI")

	client, ctx, err := connectDB(db_uri)
	checkErr(err)

	db := client.Database("mooc")
	defer client.Disconnect(ctx)
	if err := client.Database("admin").RunCommand(context.TODO(), bson.D{{"ping", 1}}).Err(); err != nil {
		panic(err)
	}
	fmt.Println("Pinged your deployment. You successfully connected to MongoDB!")

	app := fiber.New()

	app.Static("/public", "./")

	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowHeaders: "Origin, Content-Type, Accept",
	}))

	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendStatus(200)
	})

	app.Get("/docs/*", swagger.HandlerDefault)

	app.Get("/debug/:colName", func(c *fiber.Ctx) error {
		if !contextChecker(c) { return errors.New("CONTEXT IS NIL") }
		colName := c.Params("colName")
		collection := db.Collection(colName)
		rst, err := collection.Find(ctx, bson.M{})
		checkErr(err)
		return c.JSON(rst)
	})

	// history, bookmark 정보들 다 있음
	app.Post("/user/my_info", func(c *fiber.Ctx) error {
		if !contextChecker(c) { return errors.New("CONTEXT IS NIL") }
		return getMyInfo(c, ctx, db)
	})
	
	app.Post("/user/update", func(c *fiber.Ctx) error {
		if !contextChecker(c) { return errors.New("CONTEXT IS NIL") }
		return updateUser(c, ctx, db)
	})

	app.Post("/video/create", func(c *fiber.Ctx) error {
		if !contextChecker(c) { return errors.New("CONTEXT IS NIL") }
		return createVideo(c, ctx, db)
	})

	app.Get("/video/all", func(c *fiber.Ctx) error {
		if !contextChecker(c) { return errors.New("CONTEXT IS NIL") }
		return getAllVideos(c, ctx, db)
	})

	app.Get("/video/user/:id", func(c *fiber.Ctx) error {
		if !contextChecker(c) { return errors.New("CONTEXT IS NIL") }
		return getMyVideos(c, ctx, db)
	})

	app.Post("/video/delete", func(c *fiber.Ctx) error {
		if !contextChecker(c) { return errors.New("CONTEXT IS NIL") }
		return deleteVideo(c, ctx, db)
	})

	app.Get("/video/info/:video_id", func(c *fiber.Ctx) error {
		if !contextChecker(c) { return errors.New("CONTEXT IS NIL") }
		return getVideoInfo(c, ctx, db)
	})

	app.Post("/login", func(c *fiber.Ctx) error {
		if !contextChecker(c) { return errors.New("CONTEXT IS NIL") }
		return login(c, ctx, db)
	})

	app.Post("/register", func(c *fiber.Ctx) error {
		if !contextChecker(c) { return errors.New("CONTEXT IS NIL") }
		return registerUser(c, ctx, db)
	})

	app.Post("/video/comment", func(c *fiber.Ctx) error {
		if !contextChecker(c) { return errors.New("CONTEXT IS NIL") }
		return addComment(c, ctx, db)
	})

	app.Listen(":3000")
}