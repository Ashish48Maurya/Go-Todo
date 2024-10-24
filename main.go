package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Todo struct {
	ID        primitive.ObjectID `json:"_id,omitempty" bson:"_id,omitempty"`
	Completed bool               `json:"completed"`
	Desc      string             `json:"desc"`
}

var collection *mongo.Collection

func main() {

	//Require only when using locally
	if os.Getenv("ENV") != "production" {
		err := godotenv.Load(".env")
		if err != nil {
			log.Fatal("Error loading .env file:", err)
		}
	}

	// db.js code starts here
	MONGODB_URI := os.Getenv("MONGODB_URI")
	clientOptions := options.Client().ApplyURI(MONGODB_URI)
	client, err := mongo.Connect(context.Background(), clientOptions)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Disconnect(context.Background())
	err = client.Ping(context.Background(), nil)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Connection Successfull...")
	collection = client.Database("golang_db").Collection("todos")
	// db.js code ends here

	//Create server code starts here  //similar to express.js
	app := fiber.New()

	app.Use(cors.New(cors.Config{
		AllowOrigins:     "*",
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization",
		AllowCredentials: true,
	}))

	app.Get("/", backendLive)
	app.Get("/api/todos/:id", getTodos)
	app.Get("/api/todos", getTodos)
	app.Post("/api/todos", createTodo)
	app.Patch("/api/todos/:id", updateTodo)
	app.Delete("/api/todos/:id", deleteTodo)

	port := os.Getenv("PORT")
	log.Fatal(app.Listen("0.0.0.0:" + port))
	//Create server code ends here
}

func backendLive(c *fiber.Ctx) error {
	return c.Status(200).JSON(fiber.Map{"Status": "Backend is Live 🎉🎉🎉"})
}

func getTodos(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		var todos []Todo

		cursor, err := collection.Find(context.Background(), bson.M{})

		if err != nil {
			return err
		}

		defer cursor.Close(context.Background())

		for cursor.Next(context.Background()) {
			var todo Todo
			if err := cursor.Decode(&todo); err != nil {
				return err
			}
			todos = append(todos, todo)
		}

		if len(todos) == 0 {
			return c.Status(200).JSON(fiber.Map{"data": "Todos Not Available"})
		}
		return c.JSON(todos)
	}

	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid todo ID"})
	}

	filter := bson.M{"_id": objectID}
	var data Todo
	err = collection.FindOne(context.Background(), filter).Decode(&data)
	if err != nil {
		return err
	}

	return c.Status(200).JSON(data)
}

func createTodo(c *fiber.Ctx) error {
	todo := new(Todo)
	// {id:0,completed:false,body:""}

	if err := c.BodyParser(todo); err != nil {
		return err
	}

	if todo.Desc == "" {
		return c.Status(400).JSON(fiber.Map{"error": "Todo Description cannot be empty"})
	}

	insertResult, err := collection.InsertOne(context.Background(), todo)
	if err != nil {
		return err
	}
	todo.ID = insertResult.InsertedID.(primitive.ObjectID)
	return c.Status(201).JSON(fiber.Map{"data": todo, "message": "Todo Created Successfully"})
}

func updateTodo(c *fiber.Ctx) error {
	id := c.Params("id")
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid todo ID"})
	}

	var body map[string]interface{}
	//The keys are of type string, typically representing field names from, for example, a JSON object.
	//The values can be of any type (interface{}), meaning that the map can store heterogeneous data types.
	if err := c.BodyParser(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
	}
	updateFields := bson.M{}
	if completed, exists := body["completed"]; exists {
		updateFields["completed"] = completed
	}
	if desc, exists := body["desc"]; exists {
		updateFields["desc"] = desc
	}

	if len(updateFields) == 0 {
		return c.Status(400).JSON(fiber.Map{"error": "No valid fields to update"})
	}
	update := bson.M{"$set": updateFields}
	filter := bson.M{"_id": objectID}
	var updatedTodo Todo
	err = collection.FindOneAndUpdate(context.Background(), filter, update).Decode(&updatedTodo)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Todo not found or failed to update"})
	}
	return c.Status(200).JSON(fiber.Map{"data": updatedTodo, "message": "Todo Updated Successfully"})
}

func deleteTodo(c *fiber.Ctx) error {
	id := c.Params("id")
	objectID, err := primitive.ObjectIDFromHex(id)

	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid todo ID"})
	}
	filter := bson.M{"_id": objectID}
	_, err = collection.DeleteOne(context.Background(), filter)
	if err != nil {
		return err
	}
	return c.Status(200).JSON(fiber.Map{"message": "Todo Deleted Successfully"})
}
