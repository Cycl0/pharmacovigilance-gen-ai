package main

import (
  "io"
  "encoding/binary"
	"net"
  "regexp"
  "errors"
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
  "golang.org/x/exp/slices"
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
  mongoClient       *mongo.Client
	postsColl         *mongo.Collection
	medicationsColl   *mongo.Collection
)

func initDB() {
	clientOptions := options.Client().ApplyURI(os.Getenv("MONGODB_URI"))
	client, err := mongo.Connect(context.TODO(), clientOptions)
	if err != nil {
		log.Fatal(err)
	}
	mongoClient = client
	postsColl = mongoClient.Database("bluesky_data").Collection("posts")
  medicationsColl = mongoClient.Database("bluesky_data").Collection("medications")

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

// Deepseek API
func generateTextDeepSeek(prompt string) (string, error) {
	client := &http.Client{}
	reqBody := map[string]interface{}{
		"model": "deepseek-chat",
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
		"temperature": 0.7,
		"max_tokens":  500,
	}
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request body: %w", err)
	}

	req, err := http.NewRequest("POST", "https://api.deepseek.com/v1/chat/completions", bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	apiKey := os.Getenv("DEEPSEEK_API_KEY")
	if apiKey == "" {
		return "", errors.New("DEEPSEEK_API_KEY environment variable not set")
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	client = &http.Client{
		Timeout: 20 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error,omitempty"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("Failed to decode API response: %w", err)
	}

	if result.Error != nil && result.Error.Message != "" {
		return "", fmt.Errorf("API returned error: %s", result.Error.Message)
	}

	if len(result.Choices) == 0 {
		return "", errors.New("No choices returned from API")
	}

	return result.Choices[0].Message.Content, nil
}


// OpenRouter API
func generateTextOpenRouter(prompt string) (string, error) {
	client := &http.Client{}
	reqBody := map[string]interface{}{
		"model": "google/gemini-2.0-flash-001",
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
		"temperature": 0.7,
		"max_tokens":  500,
	}
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request body: %w", err)
	}

	req, err := http.NewRequest("POST", "https://openrouter.ai/api/v1/chat/completions", bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	apiKey := os.Getenv("OPENROUTER_API_KEY")
	if apiKey == "" {
		return "", errors.New("OPENROUTER_API_KEY environment variable not set")
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	client = &http.Client{
		Timeout: 20 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error,omitempty"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("Failed to decode API response: %w", err)
	}

	if result.Error != nil && result.Error.Message != "" {
		return "", fmt.Errorf("API returned error: %s", result.Error.Message)
	}

	if len(result.Choices) == 0 {
		return "", errors.New("No choices returned from API")
	}

	return result.Choices[0].Message.Content, nil
}


// OpenAI API
func generateTextOpenAI(prompt string) (string, error) {
	client := &http.Client{}
	reqBody := map[string]interface{}{
		"model": "gpt-4o-mini",
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
		"temperature": 0.7,
		"max_tokens":  500,
	}
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request body: %w", err)
	}

	req, err := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return "", errors.New("OPENAI_API_KEY environment variable not set")
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	client = &http.Client{
		Timeout: 20 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error,omitempty"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("Failed to decode API response: %w", err)
	}

	if result.Error != nil && result.Error.Message != "" {
		return "", fmt.Errorf("API returned error: %s", result.Error.Message)
	}

	if len(result.Choices) == 0 {
		return "", errors.New("No choices returned from API")
	}

	return result.Choices[0].Message.Content, nil
}

func generateTextLocalLLM(prompt string) (string, error) {
  // DEEP_THINKING_INSTRUCTION := "Enable deep thinking subroutine."
  client := &http.Client{}
	reqBody := map[string]interface{}{
		// "model": "/models/FuseO1-DeepSeekR1-QwQ-SkyT1-32B-Preview.i1-Q4_K_M.gguf",
    // "model": "/models/qwen2.5-7b-instruct-q8/qwen2.5-7b-instruct-q8_0-00001-of-00003.gguf",
    // "model": "../unsloth/output/Cogito-llama-3B-fine-tuned-pharmacovigilance.gguf",
    "model": "../unsloth/output/Qwen2.5-7B-Instruct-LoRA-fine-tuned-pharmacovigilance.gguf",

		"messages": []map[string]string{
      // {"role": "system", "content": DEEP_THINKING_INSTRUCTION},
			{"role": "user", "content": prompt},
		},
		"temperature": 0.7,
		"max_tokens":  1024,
	}

	jsonBody, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", "http://127.0.0.1:8000/v1/chat/completions", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")

  // Cliente customizado com 20 segundos de timeout
  client = &http.Client{
    Timeout: 20 * time.Second,
  }

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Llama API error: %v", err)
		return "", errors.New("Llama API error")
	}
	defer resp.Body.Close()

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error,omitempty"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("Failed to decode API response: %w", err)
	}

	if result.Error != nil && result.Error.Message != "" {
		return "", fmt.Errorf("API returned error: %s", result.Error.Message)
	}

	if len(result.Choices) > 0 {
    message := result.Choices[0].Message.Content

    fmt.Printf("%s\n", message)

    // Remover <think> tags e o conteudo dentro
    re := regexp.MustCompile(`(?s)<think>.*?</think>`)
    cleaned := re.ReplaceAllString(message, "")
    // Tira espacos em branco se sobrar
    cleaned = strings.TrimSpace(cleaned)
    cleaned = regexp.MustCompile(`\n{3,}`).ReplaceAllString(cleaned, "\n")

		return cleaned, nil
	}
	return "", errors.New("No choices returned from API")
}

func generateTextUmbrella(prompt string) (string, error) {
    conn := connectToServer()
    defer conn.Close()

    // First confirm we can receive the welcome message
    welcomeLength := make([]byte, 4)
    if _, err := io.ReadFull(conn, welcomeLength); err != nil {
        return "", fmt.Errorf("error reading welcome length: %v", err)
    }

    welcomeSize := binary.BigEndian.Uint32(welcomeLength)
    welcomeData := make([]byte, welcomeSize)
    if _, err := io.ReadFull(conn, welcomeData); err != nil {
        return "", fmt.Errorf("error reading welcome data: %v", err)
    }

    fmt.Printf("Welcome message: %s\n", string(welcomeData))

    // Simple request with only required fields
    req := APIRequest{
        Context:      prompt,
        MaxNewTokens: 512,
        Temperature:  0.7,
    }

    // Send request and get response
    responseText, err := sendRequest(conn, req)
    if err != nil {
        return "", fmt.Errorf("request failed: %w", err)
    }

    // Parse the response to extract the generated text
    var response map[string]interface{}
    if err := json.Unmarshal([]byte(responseText), &response); err != nil {
        return "", fmt.Errorf("failed to parse response: %w", err)
    }

    // Extract the generated text from the "generated_text" field
    generatedText, ok := response["generated_text"].(string)
    if !ok {
        return "", fmt.Errorf("could not find generated text in response")
    }

    // Try to send termination request but handle EOF gracefully
    _, err = sendRequest(conn, APIRequest{
        MaxNewTokens: 0,
        Temperature: 0,
        Terminate:   true,
    })

    // Just log termination errors rather than failing the whole function
    if err != nil {
        log.Printf("Warning: termination request error: %v", err)
    }

    return generatedText, nil
}

func sendRequest(conn net.Conn, req APIRequest) (string, error) {
    // Serialize using JSON
    data, err := json.Marshal(req)
    if err != nil {
        return "", fmt.Errorf("JSON marshaling error: %v", err)
    }

    // Debug - print what we're sending
    fmt.Printf("Sending data: %s\n", string(data))

    // Send length prefix
    length := make([]byte, 4)
    binary.BigEndian.PutUint32(length, uint32(len(data)))
    if _, err := conn.Write(length); err != nil {
        return "", fmt.Errorf("failed to write length: %v", err)
    }

    // Send payload
    if _, err := conn.Write(data); err != nil {
        return "", fmt.Errorf("failed to write data: %v", err)
    }

    // Receive response
    responseLength := make([]byte, 4)
    if _, err = io.ReadFull(conn, responseLength); err != nil {
        return "", fmt.Errorf("error reading response length: %v", err)
    }

    responseSize := binary.BigEndian.Uint32(responseLength)
    responseData := make([]byte, responseSize)
    if _, err = io.ReadFull(conn, responseData); err != nil {
        return "", fmt.Errorf("error reading response data: %v", err)
    }

    fmt.Printf("Received response: %s\n", string(responseData))
    return string(responseData), nil
}

type APIRequest struct {
	Context        string `json:"context,omitempty"`
	InputIDs       []int  `json:"input_ids,omitempty"`
	MaxNewTokens   int    `json:"max_new_tokens"`
	Temperature    float64 `json:"temperature"`
	Terminate      bool   `json:"terminate,omitempty"`
}

func connectToServer() net.Conn {
	retryInterval := 5 * time.Second
	maxRetries := 5

	for i := 0; i < maxRetries; i++ {
		conn, err := net.Dial("tcp", "localhost:65432")
		if err == nil {
			log.Println("Connected to server")
			return conn
		}

		log.Printf("Connection failed (attempt %d/%d): %v", i+1, maxRetries, err)
		time.Sleep(retryInterval)
	}
	log.Fatal("Failed to connect after multiple attempts")
	return nil
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

type Medication struct {
    Name string
    ADRs []string
}

func parseMedications(input string, query string) []Medication {
    var medications []Medication
    entries := strings.Split(input, "|")

    for _, entry := range entries {
        parts := strings.Split(entry, ",")
        if len(parts) < 1 {
            continue // pula entradas vazias
        }

        for i := range parts {
            parts[i] = strings.TrimSpace(parts[i])
        }

        medicine := parts[0]
        if medicine == "X" {
          medicine = query
        }

        med := Medication{
            Name: medicine,
            ADRs: parts[1:],
        }

        medications = append(medications, med)
    }

    return medications
}

func main() {
  benchmarkTime := time.Now();

  var queryList = [...]string{"Venvanse", "Aripiprazol", "Fluoxetina", "Escitalopram", "Sertralina", "Ritalina", "Atentah", "Concerta", "Bupropiona", "Risperidona", "Paroxetina", "Venlafaxina", "Vortioxetina",  "Agomelatina", "Desvenlafaxina", "Duloxetina", "Vortioxetina", "Nefazodona", " Trazodona", "Clonazepam", "Alprazolam", "Lorazepam", "Bromazepam", "Diazepam", "Amitriptilina", "Clomipramina", "Desipramina", "Doxepina", "Imipramina", "Maprotilina", "Nortriptilina", "Protriptilina", "Trimipramina", "Puran", "Salonpas", "Cliclo", "Microvlar", "Buscopan", "Rivotril", "Dorflex", "Glifage"}
  var adrList = []string{"Nausea", "Apathy", "Anxiety", "Sleepiness", "Arrhythmia"}

  for _, query := range queryList {
    initDB()
    accessToken := authenticate()
    client := &http.Client{}
    maxResults := 500
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

      log.Printf("API: %d posts, cursor=%v, Total=%d",
    len(result.Posts), result.Cursor != "", totalRetrieved)

      for _, post := range result.Posts {

        // Prompt 1

        // prompt := fmt.Sprintf(`Answer with the side effects in english for yes and X for no.
        // 	DO NOT EXPLAIN OR COMMENT
        // 	The answer MUST above all be a single character or a list of just the name of the side effect without any aditional commentary/information/detail or an X for when this is not applied.
        // 	Does this text talk about %s and its side effects.
        // 	If the text only talks about %s and the symptoms that afflicts them but aren't specifically side effects from %s, answer with no (X).
        // 	Answer with the main side effects in english only if the side effects are from %s and they are bad.
        //   If there are multiple side effects, separate them with a single comma without any whitespace
        //   Text: %s`, query, query, query, query, post.Record.Text)

        // Prompt 2

        // prompt := fmt.Sprintf(`
        // You are a pharmacovigilance specialist and you are analyzing the side effects regarding %s in social media posts.
        // Answer with the side effects in english if there are any and X for no.
        // YOU MUST BE ABLE TO UNDERSTAND AND INTERPRET INFORMAL LANGUAGE IN ANY LANGUAGE, YOU MUST NOT CONFUSE SIDE EFFECTS WITH THE SYMPTHOMS THE MEDICINE SOLVES OR GIVES WHEN ONE STOPS TAKING IT
        // YOU MUST NOT ASSUME THE WHAT THE SIDE EFFECTS ARE, YOU SHOULD EXTRACT IT FROM THE TEXT
        // DO NOT EXPLAIN OR COMMENT
        // The answer MUST above all be a single character or a list of just the name of the side effect without any aditional commentary/information/detail or an X for when this is not applied.
        // Does this post talk about %s and its side effects, physical or emotional?
        // If the text only talks about %s and the symptoms that afflicts them but aren't specifically side effects from %s, answer with no (X).
        // Answer with the main side effects in english only if the side effects are from %s and they are bad or undesirable.
        // If there are multiple side effects, separate them with a single comma without any whitespace
        // Post: %s`, query, query, query, query, post.Record.Text)


        // Prompt 3

        // prompt := fmt.Sprintf(`

        //   Only answer in english in a single line with the output following these templates
        //   medicine is always first
        //   (adr is adverse drug reaction)
        //   replace each one with the actual medicine and the actual respective adrs
        //   if an adr is non existent, put an upper case X instead
        //   Each list has a head (the first element), the head will always be the medicine name and the rest will be the adrs
        //   USE the following separator ":" to separate the lists
        //   <medicine1>,<adr1>:<medicine2>,<adr1>

        //   DO NOT DEVIATE FROM THE OUTPUT TEMPLATE
        //   Example 1 of output:
        //   <medicine1>,<adr1>
        //   Example 2 of output:
        //   <medicine1>,<adr1>,<adr2>,<adr3>
        //   Example 3 of output:
        //   <medicine1>,<adr1>,<adr2>:<medicine1>,<adr1>,<adr2>,<adr3>
        //   Example 4 of outupt:
        //   <medicine1>,<adr1>:<medicine2>,<adr1>:<medicine3>,<adr1>,<adr2>,<adr3>

        //   You are a pharmacovigilance specialist and you are analyzing the side effects regarding medicines in social media posts.
        //   YOU MUST BE ABLE TO UNDERSTAND AND INTERPRET INFORMAL LANGUAGE IN ANY LANGUAGE, YOU MUST NOT CONFUSE SIDE EFFECTS WITH THE SYMPTHOMS THE MEDICINE SOLVES
        //   YOU MUST NOT ASSUME  WHAT THE SIDE EFFECTS ARE, YOU SHOULD EXTRACT IT FROM THE TEXT AND RESUME IT
        // 	DO NOT EXPLAIN OR COMMENT
        // 	Does this post talk about a medicine and its side effects, physical or emotional?
        //   Put an X in the first adr field for the respective medicine if it's talking about sympthons that are not related to the medicine
        // 	Translate to english the main side effects each resumed in a one or two words and the name of the medicine
        //   Post: %s`, post.Record.Text)


        // Prompt 4
        prompt := fmt.Sprintf(`

          Only answer in english in a single line with the output following these templates
          medicine is always first
          (adr is adverse drug reaction)
          replace each one with the actual medicine and the actual respective adrs
          Each list has a head (the first element), the head will always be the medicine name and the rest will be the adrs
          USE the following separator "|" to separate the lists like in:
          medicine1,adr1|medicine2,adr1

          Example 1 of output if there is a single medicine with a single adr: medicine1,adr1
          Example 2 of output: medicine1,adr1,adr2,adr3
          Example 3 of output with multiple medicines and multiple adrs: medicine1,adr1,adr2|medicine1,adr1,adr2,adr3,adr4
          Example 4 of outupt: medicine1,adr1|medicine2,adr1|medicine3,adr1,adr2,adr3
          Example 5 of output if there is just a medicine: medicine1
          Example 6 of output if theree is just adrs and no medicine: X,adr1,adr2,adr3

          So if the Post was: 'Fluoxetina me da nausea e apatia, Venvanse me deixa ansiosa'
          The output would be for example (DO NOT COPY THIS IS AN EXAMPLE):
          Fluoxetine,Nausea,Apathy|Venvanse,Anxiety

          BUT ONLY DO THAT IF THE USER IS TALKING ABOUT A MEDICINE AND THEIR SIDE EFFECTS, PUT JUST THE NAME OF THE MEDICINE IF THAT IS NOT THE CASE
          CAPTURE THE NAMES OF THE MEDICINES AND THEIR ADVERSE REACTIONS RESUMED, DO NOT CAPTURE ANYTHING ELSE
          AVOID AT ALL COSTS NOTES, OBSERVATIONS OR ANY COMMENTARY

          Does this post talk about a medicine and its side effects, physical or emotional?

          USE THIS LIST AS REFERENCE FOR THE ADRS: %S. ONLY DEVIATE FROM THE LIST IF THE ADR IS NOT ABSOLUTELY NOT PRESENT ON THE LIST FOR EXAPLE SOMNOLENCE IS THE SAME AS SLEEPINESS SO SLEEPINESS SHOULD BE USED

          Post: %s`, strings.Join(adrList, ","), post.Record.Text)

      print(fmt.Sprintf("\n***\nADRs Lista: %s\n***\n", strings.Join(adrList, ",")))


      answer, errGeneration := generateTextOpenRouter(prompt)
        if errGeneration != nil {
          log.Printf("Error: %v", errGeneration)
        }

        analysis := parseMedications(answer,query)

        var medicationUpdates []mongo.WriteModel
        for _, med := range analysis {
          if med.Name == "" {
            continue
          }
          // Filtrar fora 'X' (Que significa sem ADRs)
          filteredADRs := make([]string, 0)
          for _, adr := range med.ADRs {
            if adr != "X" {
              filteredADRs = append(filteredADRs, adr)
              if !slices.Contains(adrList, adr) {
                adrList = append(adrList, adr)
              }
            }
          }

          if len(filteredADRs) == 0 {
            continue
          }

          // Filtro Case-insensitive
          filter := bson.M{
            "name": bson.M{
              "$regex":   "^" + regexp.QuoteMeta(med.Name) + "$",
              "$options": "i", // Case-insensitive
            },
          }

          update := bson.M{
            "$addToSet": bson.M{
              "adrs": bson.M{"$each": filteredADRs},
            },
            "$inc": bson.M{"mentionCount": 1},
            "$setOnInsert": bson.M{
              "name":          med.Name,
              "firstMentioned": primitive.NewDateTimeFromTime(time.Now().UTC()),
            },
          }

          model := mongo.NewUpdateOneModel().
            SetFilter(filter).
            SetUpdate(update).
            SetUpsert(true)

          medicationUpdates = append(medicationUpdates, model)
        }

        if len(medicationUpdates) > 0 {
          _, err := medicationsColl.BulkWrite(context.TODO(), medicationUpdates)
          if err != nil {
            log.Printf("Medication update error: %v", err)
          }
        }

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
          "query": query,
          "rawOutput": answer,
          "analysis": analysis,
        })
        _, err := postsColl.InsertMany(context.TODO(), documents, options.InsertMany().SetOrdered(true))
        if err != nil {
          log.Printf("Insert error: %v", err)
        }

        totalRetrieved += 1
        fmt.Printf("Posts verificados: %d\n---\n\n", totalRetrieved)

        print(fmt.Sprintf(`Usuario: %s

  `, post.Author.DisplayName))
        print(fmt.Sprintf(`Texto: %s

  `, post.Record.Text))
        print(fmt.Sprintf(`Output: %s
  `, answer))
        print(fmt.Sprintf(`Analise: %s

  ---

  `, analysis))

      }

      cursor = result.Cursor

      if cursor == "" || totalRetrieved >= maxResults {
        timeElapsed := time.Since(benchmarkTime)
        print(fmt.Sprintf("\n--- Tempo total: %s ---\n", timeElapsed))
        break
      }
      time.Sleep(1 * time.Second)
    }
  }
}
