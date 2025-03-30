package main

import (
  "bytes"
	"context"
	"encoding/json"
	"fmt"
  "strings"
	"log"
	"net/http"
	"os"
	"time"
	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// .env
func init() {
	if err := godotenv.Load("/app/.env"); err != nil {
		log.Fatal("Error loading .env file")
	}
}

// Structs para BlueSky API
type Post struct {
	URI    string `json:"uri"`
	Author struct {
		DID         string `json:"did"`
		Handle      string `json:"handle"`
		DisplayName string `json:"displayName"`
	} `json:"author"`
	Record struct {
		Text      string    `json:"text"`
		CreatedAt string    `json:"createdAt"`
	} `json:"record"`
}

type SessionResponse struct {
	AccessJWT string `json:"accessJwt"`
	DID       string `json:"did"`
}

// MongoDB
var (
	mongoClient *mongo.Client
	postsColl   *mongo.Collection
)

func initDB() {
	clientOptions := options.Client().ApplyURI(os.Getenv("MONGODB_URI"))
	client, err := mongo.Connect(context.TODO(), clientOptions)
	if err != nil {
		log.Fatal(err)
	}
	mongoClient = client
	postsColl = mongoClient.Database("bluesky_data").Collection("posts")

	// Index unico
	indexModel := mongo.IndexModel{
		Keys:    bson.D{{Key: "post_uri", Value: 1}},
		Options: options.Index().SetUnique(true),
	}
	_, err = postsColl.Indexes().CreateOne(context.TODO(), indexModel)
	if err != nil {
		log.Fatal(err)
	}
}

// DeepSeek API
func generateTextDeepSeek(prompt string) string {
	client := &http.Client{}
	reqBody := map[string]interface{}{
		"model": "deepseek-chat",
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
		"temperature": 0.7,
		"max_tokens":  500,
	}

	jsonBody, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", "https://api.deepseek.com/v1/chat/completions", bytes.NewBuffer(jsonBody))
	req.Header.Set("Authorization", "Bearer "+os.Getenv("DEEPSEEK_API_KEY"))
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("DeepSeek API error: %v", err)
		return ""
	}
	defer resp.Body.Close()

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	json.NewDecoder(resp.Body).Decode(&result)

	if len(result.Choices) > 0 {
		return result.Choices[0].Message.Content
	}
	return ""
}

// OpenAI API
func generateTextOpenAI(prompt string) string {
	client := &http.Client{}
	reqBody := map[string]interface{}{
		"model": "gpt-4o-mini",
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
		"temperature": 0.7,
		"max_tokens":  500,
	}

	jsonBody, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(jsonBody))
	req.Header.Set("Authorization", "Bearer "+os.Getenv("OPENAI_API_KEY"))
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("OpenAI API error: %v", err)
		return ""
	}
	defer resp.Body.Close()

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	json.NewDecoder(resp.Body).Decode(&result)

	if len(result.Choices) > 0 {
		return result.Choices[0].Message.Content
	}
	return ""
}

// BlueSky autenticacao
func authenticate() string {
	reqBody := map[string]string{
		"identifier": os.Getenv("BLUESKY_USERNAME"),
		"password":   os.Getenv("BLUESKY_APP_PASSWORD"),
	}
	jsonBody, _ := json.Marshal(reqBody)

	resp, err := http.Post(
		"https://bsky.social/xrpc/com.atproto.server.createSession",
		"application/json",
		bytes.NewBuffer(jsonBody),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	var session SessionResponse
	json.NewDecoder(resp.Body).Decode(&session)
	return session.AccessJWT
}

func main() {
  benchmarkTime := time.Now();

	initDB()
	accessToken := authenticate()
	searchTerm := "Fluoxetina"

	client := &http.Client{}
	query := searchTerm
	maxResults := 1043
	totalRetrieved := 0
	cursor := ""

	for {
		req, _ := http.NewRequest("GET", "https://bsky.social/xrpc/app.bsky.feed.searchPosts", nil)
		q := req.URL.Query()
		q.Add("q", query)
		q.Add("limit", "100")
		if cursor != "" {
			q.Add("cursor", cursor)
		}
		req.URL.RawQuery = q.Encode()
		req.Header.Add("Authorization", "Bearer "+accessToken)

		resp, err := client.Do(req)
		if err != nil {
			log.Fatal(err)
		}

		var result struct {
			Posts  []Post `json:"posts"`
			Cursor string `json:"cursor"`
		}
		json.NewDecoder(resp.Body).Decode(&result)
		resp.Body.Close()

		var relevantPosts []Post
		for _, post := range result.Posts {

      // Prompt 1

      // prompt := fmt.Sprintf(`Answer with the side effects in english for yes and X for no.
			// 	DO NOT EXPLAIN OR COMMENT
			// 	The answer MUST above all be a single character or a list of just the name of the side effect without any aditional commentary/information/detail or an X for when this is not applied.
			// 	Does this text talk about %s and its side effects.
			// 	If the text only talks about %s and the symptoms that afflicts them but aren't specifically side effects from %s, answer with no (X).
			// 	Answer with the main side effects in english only if the side effects are from %s and they are bad.
      //   If there are multiple side effects, separate them with a single comma without any whitespace
      //   Text: %s`, searchTerm, searchTerm, searchTerm, searchTerm, post.Record.Text)

      // Prompt 2

			prompt := fmt.Sprintf(`
        You are a pharmacovigilance specialist and you are analyzing a the side effects regarding %s in social media posts.
        Answer with the side effects in englis if there is and X for no.
        YOU MUST BE ABLE TO UNDERSTAND AND INTERPRET INFORMAL LANGUAGE IN ANY LANGUAGE, YOU MUST NOT CONFUSE SIDE EFFECTS WITH THE SYMPTHOMS THE MEDICINE SOLVES OR GIVES WHEN ONE STOPS TAKING IT
        YOU MUST NOT ASSUME THE WHAT THE SIDE EFFECTS ARE, YOU SHOULD EXTRACT IT FROM THE TEXT
				DO NOT EXPLAIN OR COMMENT
				The answer MUST above all be a single character or a list of just the name of the side effect without any aditional commentary/information/detail or an X for when this is not applied.
				Does this post talk about %s and its side effects, physical or emotional?
				If the text only talks about %s and the symptoms that afflicts them but aren't specifically side effects from %s, answer with no (X).
				Answer with the main side effects in english only if the side effects are from %s and they are bad or undesirable.
        If there are multiple side effects, separate them with a single comma without any whitespace
        Post: %s`, searchTerm, searchTerm, searchTerm, searchTerm, searchTerm, post.Record.Text)


			sideEffects := generateTextOpenAI(prompt)
			if sideEffects != "X" && sideEffects != "" {
				relevantPosts = append(relevantPosts, post)
        createdAt, _ := time.Parse(time.RFC3339Nano, post.Record.CreatedAt)
        var documents []interface{}
				documents = append(documents, bson.M{
					"post_uri": post.URI,
					"author": bson.M{
						"did":          post.Author.DID,
						"handle":       post.Author.Handle,
						"display_name": post.Author.DisplayName,
					},
					"content":      post.Record.Text,
					"created_at":   primitive.NewDateTimeFromTime(createdAt),
					"indexed_at":   primitive.NewDateTimeFromTime(time.Now().UTC()),
					"search_query": searchTerm,
          "side_effects": strings.Split(sideEffects, ","),
				})
        _, err := postsColl.InsertMany(context.TODO(), documents, options.InsertMany().SetOrdered(false))
        if err != nil {
            log.Printf("Insert error: %v", err)
        }
			}

      totalRetrieved += 1
      fmt.Printf("Posts verificados: %d\n---\n\n", totalRetrieved)

      print(fmt.Sprintf(`Usuario: %s
`, post.Author.DisplayName))
      print(fmt.Sprintf(`Texto: %s
`, post.Record.Text))
      print(fmt.Sprintf(`Efeitos colaterais: %s

---

`, sideEffects))

      if result.Cursor == "" || totalRetrieved >= maxResults {
        timeElapsed := time.Since(benchmarkTime)
        print(fmt.Sprintf("\n--- Tempo total: %s ---\n", timeElapsed))
        break
      }

		}

    if result.Cursor == "" || totalRetrieved >= maxResults {
      break
    }
    cursor = result.Cursor
    time.Sleep(1 * time.Second)
	}
}
