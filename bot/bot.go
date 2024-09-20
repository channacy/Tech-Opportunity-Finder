package bot

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"webscraper/shared"

	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/bwmarrin/discordgo"
)

func StartBot() {
	envFile, _ := godotenv.Read(".env")

	discordToken := envFile["DISCORD_TOKEN"]
	mongoURI := envFile["MONGODB_URI"]

	result := getMessage(mongoURI)
	message := convertToString(result)
	sess, err := discordgo.New("Bot " + discordToken)
	if err != nil {
		log.Fatal(err)
	}

	sess.AddHandler(func(s *discordgo.Session, m *discordgo.MessageCreate) {
		//respond to any messages
		if m.Author.ID == s.State.User.ID {
			return
		}

		if m.Content == "hello" {
			s.ChannelMessageSend(m.ChannelID, "world!")
		}

		if m.Content == "test" {
			s.ChannelMessageSend(m.ChannelID, message)
		}
	})

	//send intent to server -> send info to us
	sess.Identify.Intents = discordgo.IntentsAllWithoutPrivileged

	err = sess.Open()
	if err != nil {
		log.Fatal(err)
		fmt.Println("error opening connection")
	}
	defer sess.Close()

	fmt.Println("Tech Opportunity bot is online! Press CTRL+C to exit.")

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

}

func getMessage(mongoURI string) (result []shared.Opportunity) {
	serverAPI := options.ServerAPI(options.ServerAPIVersion1)
	opts := options.Client().ApplyURI(mongoURI).SetServerAPIOptions(serverAPI)

	client, err := mongo.Connect(context.TODO(), opts)
	if err != nil {
		panic(err)
	}

	defer func() {
		if err = client.Disconnect(context.TODO()); err != nil {
			panic(err)
		}
	}()

	if err := client.Database("techOptFinderDB").RunCommand(context.TODO(), bson.D{{"ping", 1}}).Err(); err != nil {
		panic(err)
	}

	fmt.Println("Pinged your deployment. You successfully connected to MongoDB!")

	db := client.Database("techOptFinderDB").Collection("opportunities")

	condition := bson.M{
		"category": "Articles",
	}
	cur, err := db.Find(context.Background(), condition)
	if err != nil {
		log.Fatal(err)
	}

	var data []shared.Opportunity
	if err := cur.All(context.Background(), &data); err != nil {
		log.Fatal(err)
	}

	// now we can use the data array, which contains all of the documents
	for _, opportunity := range data {
		log.Printf("the name is %v\n", opportunity.Name)
	}

	return data
}

func convertToString(result []shared.Opportunity) string {
	var builder strings.Builder

	for i, data := range result {
		if i > 3 {
			break
		}
		// Append the person's details to the builder
		builder.WriteString(fmt.Sprintf("Name: %s, Url: %d\n", data.Name, data.Url))
	}

	//builder -> string
	return builder.String()
}
