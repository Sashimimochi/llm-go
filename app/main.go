package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/labstack/echo/v4"
	usearch "github.com/unum-cloud/usearch/golang"
)

type EmbeddingVector [][]float32

type EmbeddingResponse struct {
	Embedding []float32 `json:"embedding"`
}

type EmbeddingRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
}

type ChatRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
	Stream   bool      `json:"stream"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatResponse struct {
	Model              string    `json:"model"`
	CreatedAt          time.Time `json:"created_at"`
	Message            Message   `json:"message"`
	DoneReason         string    `json:"done_reason"`
	Done               bool      `json:"done"`
	TotalDuration      int64     `json:"total_duration"`
	LoadDuration       int       `json:"load_duration"`
	PromptEvalCount    int       `json:"prompt_eval_count"`
	PromptEvalDuration int       `json:"prompt_eval_duration"`
	EvalCount          int       `json:"eval_count"`
	EvalDuration       int64     `json:"eval_duration"`
}

const (
	defaultOllamaURL = "http://host.docker.internal:11434/api"
	modelName        = "llm"
)

func main() {
	e := echo.New()
	e.GET("/chat", func(c echo.Context) error {
		indexPath, err := createIndex()
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": fmt.Sprintf("%v", err),
			})
		}
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": fmt.Sprintf("%v", err),
			})
		}
		// RAGを実行
		prompt := c.QueryParam("prompt")
		if prompt == "" {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "promptを指定してください",
			})
		}
		log.Printf("user prompt: %v", prompt)
		ref, err := Search(indexPath, prompt)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": fmt.Sprintf("%v", err),
			})
		}
		chatResp, err := chat(prompt + "\n===\n以下の情報を下に回答してください。\n===\n" + ref)
		if err != nil {
			err_msg := fmt.Sprintf("Failed to chat: %v", err)
			log.Fatalf(err_msg)
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": err_msg,
			})
		}

		return c.JSON(http.StatusOK, map[string]string{
			"message": chatResp,
		})
	})

	// Start Server
	e.Logger.Fatal(e.Start(":8080"))
}

func readDocs() ([]string, error) {
	docFilename := "docs.txt"
	file, err := os.Open(docFilename)
	if err != nil {
		log.Fatalf("%v", err)
		return nil, err
	}
	defer file.Close()
	var texts []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		texts = append(texts, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		log.Fatalf("%v", err)
		return nil, err
	}
	return texts, nil
}

func createIndex() (string, error) {
	indexPath := "vector.index"
	if _, err := os.Stat(indexPath); err == nil {
		return indexPath, nil
	}

	texts, err := readDocs()
	if err != nil {
		return "", err
	}
	embeddings, err := createVectors(texts)
	if err != nil {
		return "", err
	}

	log.Printf("Creating Vector Index")
	// usearchでインデックスを作成
	vectorSize := len(embeddings[0])
	vectorsCount := len(embeddings)
	log.Printf("Embedding Dimension: %d, Document size: %d", vectorSize, vectorsCount)
	conf := usearch.DefaultConfig(uint(vectorSize))
	index, err := usearch.NewIndex(conf)
	if err != nil {
		err_msg := fmt.Sprintf("Failed to create Index: %v", err)
		log.Fatalf(err_msg)
		return "", fmt.Errorf(err_msg)
	}
	defer index.Destroy()

	// インデックスに追加
	err = index.Reserve(uint(vectorsCount))
	if err != nil {
		err_msg := fmt.Sprintf("Failed to reserve space for Index: %v", err)
		log.Fatalf(err_msg)
		return "", fmt.Errorf(err_msg)
	}
	for i := 0; i < vectorsCount; i++ {
		err = index.Add(usearch.Key(i), embeddings[i])
		if err != nil {
			err_msg := fmt.Sprintf("Failed to add vector to Index: %v", err)
			log.Fatalf(err_msg)
			return "", fmt.Errorf(err_msg)
		}
	}

	// インデックスを保存
	err = index.Save(indexPath)
	if err != nil {
		err_msg := fmt.Sprintf("Failed to save Index: %v", err)
		log.Fatalf(err_msg)
		return "", fmt.Errorf(err_msg)
	}
	log.Printf("Index successfully created and saved to %v", indexPath)
	return indexPath, nil
}

func createVectors(texts []string) (EmbeddingVector, error) {
	// ベクトルデータを作成
	log.Printf("Creating Embedding Vectors")
	embeddings := make(EmbeddingVector, 0)
	for _, text := range texts {
		embedding, err := embeddingText(text)
		if err != nil {
			log.Fatalf("%v", err)
			return nil, err
		}
		embeddings = append(embeddings, embedding)
	}
	return embeddings, nil
}

func chat(prompt string) (string, error) {
	log.Printf("Chating to LLM")
	const url = defaultOllamaURL + "/chat"

	msg := Message{
		Role:    "user",
		Content: prompt,
	}
	req := ChatRequest{
		Model:    modelName,
		Stream:   false,
		Messages: []Message{msg},
	}

	resp, err := talkOllama(url, req)
	if err != nil {
		return "", err
	}

	// レスポンス
	chat := resp.Message.Content
	return chat, nil
}

func talkOllama(url string, ollamaReq ChatRequest) (*ChatResponse, error) {
	js, err := json.Marshal(&ollamaReq)
	if err != nil {
		return nil, err
	}
	client := http.Client{}
	httpReq, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(js))
	if err != nil {
		return nil, err
	}
	httpResp, err := client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer httpResp.Body.Close()
	ollamaResp := ChatResponse{}
	err = json.NewDecoder(httpResp.Body).Decode(&ollamaResp)
	return &ollamaResp, err
}

func embeddingText(prompt string) ([]float32, error) {
	url := defaultOllamaURL + "/embeddings"

	// リクエストボディを作成
	req := EmbeddingRequest{
		Model:  modelName,
		Prompt: prompt,
	}
	js, err := json.Marshal(&req)

	// POSTリクエストを作成
	httpReq, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(js))
	if err != nil {
		err_msg := fmt.Sprintf("Failed to create request: %v", err)
		log.Fatalf(err_msg)
		return nil, fmt.Errorf(err_msg)
	}

	// クライアントを作成
	client := &http.Client{}
	httpResp, err := client.Do(httpReq)
	if err != nil {
		err_msg := fmt.Sprintf("Failed to make POST request: %v", err)
		log.Fatalf(err_msg)
		return nil, fmt.Errorf(err_msg)
	}
	defer httpResp.Body.Close()

	// レスポンスボディをパース
	embeddingResp := EmbeddingResponse{}
	err = json.NewDecoder(httpResp.Body).Decode(&embeddingResp)
	if err != nil {
		err_msg := fmt.Sprintf("Failed to parse response body: %v", err)
		log.Fatalf(err_msg)
		return nil, fmt.Errorf(err_msg)
	}

	// 埋め込みベクトルを取得
	embedding := embeddingResp.Embedding
	return embedding, nil
}

func Search(indexPath string, prompt string) (string, error) {
	log.Printf("Searching Nearest Neighbor Index")
	embedding, err := embeddingText(prompt)
	if err != nil {
		log.Fatalf("%v", err)
		return "", err
	}

	// usearchでインデックスを作成
	vectorSize := len(embedding)
	conf := usearch.DefaultConfig(uint(vectorSize))
	index, err := usearch.NewIndex(conf)
	if err != nil {
		err_msg := fmt.Sprintf("Failed to create Index: %v", err)
		log.Fatalf(err_msg)
		return "", fmt.Errorf(err_msg)
	}
	defer index.Destroy()

	// Search
	err = index.Load(indexPath)
	if err != nil {
		err_msg := fmt.Sprintf("Failed to load Index: %v", err)
		log.Fatalf(err_msg)
		return "", fmt.Errorf(err_msg)
	}

	keys, distances, err := index.Search(embedding, 3)
	if err != nil {
		err_msg := fmt.Sprintf("Failed to search: %v", err)
		log.Fatalf(err_msg)
		return "", fmt.Errorf(err_msg)
	}
	for i := 0; i < len(keys); i++ {
		log.Printf("key: %v, distance: %v", keys[i], distances[i])
	}
	texts, err := readDocs()
	return texts[keys[0]], nil
}
