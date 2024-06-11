package main

import (
    "bytes"
    "encoding/json"
    "fmt"
    "io/ioutil"
    "log"
    "net/http"

    usearch "github.com/unum-cloud/usearch/golang"
)

type EmbeddingResponse struct {
    Embedding []float32 `json:"embedding"`
}

type ChatResponse struct {
    Model string `json:"model"`
    CreatedAt string `json:"created_at"`
    Message struct {
        Role string `json:"role"`
        Content string `json:"content"`
    }
    DoneReason string `json:"done_reason"`
    Done bool `json:"done"`
    TotalDuration int `json:"total_duration"`
    LoadDuration int `json:"load_duration"`
    PromptEvalCount int `json:"prompt_eval_count"`
    PromptEvalDuration int `json:"prompt_eval_duration"`
    EvalCount int `json:"eval_count"`
    EvalDuration int `json:"eval_duration"`
}

func main() {
    var texts[3] string = [3]string {"RAGは、プロンプトと呼ばれるLLMへの入力情報にモデル外部の情報源からの検索結果を付加して、検索結果に基づいた回答をさせています。モデルはプロンプトに含まれているリファレンス情報をもとにユーザからの要望に回答すればいいので、知識にないことであっても正確に答えられるようになるのではないかという発想です。", "HNSWはグラフ探索型のアルゴリズムでベクトル距離に応じて事前にグラフを構築しておきます。探索は高速だが、事前に構築したグラフの読み込みに時間がかかるという特性があります。また、圧縮せずに生データをそのままメモリ上に展開して処理をするので、メモリの消費量が大きくなります。2020年にGoogleがScaNNというより高速で高精度・高効率なアルゴリズムを提唱しましたが、2018年時点ではmillion-scaleのドキュメントの近似近傍探索では決定版と言われていたアルゴリズムです。", "根幹となる文のポジティブ/ネガティブ判定を実装していきます。文のポジネガ判定の方法もいろいろあると思いますが、今回は単語ごとの印象(極性)を分析していく極性分析を行うことにします。単語ごとの極性をまとめた極性辞書は、東京工業大学精密工学研究所高村研究室が公開している単語感情極性対応表を使いました。表の中身は単語ごとに-1~1までの極性値が載っています。これに文を単語分割したものを渡してポジネガ値を計算することにします。"}
    embeddings := make([][]float32, 0)

    for _, text := range texts {
        embedding, err := embeddingText(text)
        if err != nil {
            log.Fatalf("%v", err)
        }
        embeddings = append(embeddings, embedding)
    }
    // usearchでインデックスを作成
    vectorSize := len(embeddings[0])
    vectorsCount := len(embeddings)
    log.Printf("Embedding Dimension: %d, Document size: %d", vectorSize, vectorsCount)
    conf := usearch.DefaultConfig(uint(vectorSize))
    index, err := usearch.NewIndex(conf)
    if err != nil {
        log.Fatalf("Failed to create Index: %v", err)
    }
    defer index.Destroy()

    // インデックスに追加
    err = index.Reserve(uint(vectorsCount))
    if err != nil {
        log.Fatalf("Failed to reserve space for Index: %v", err)
    }
    for i := 0; i < vectorsCount; i++ {
        err = index.Add(usearch.Key(i), embeddings[i])
        if err != nil {
            log.Fatalf("Failed to add vector to Index: %v", err)
        }    
    }

    // インデックスを保存
    indexPath := "vector.index"
    err = index.Save(indexPath)
    if err != nil {
        log.Fatalf("Failed to save Index: %v", err)
    }
    log.Printf("Index successfully created and saved to %v", indexPath)

    Search(indexPath)

    chatResp, err := chat("RAGという検索手法について完結に教えてください。")
    if err != nil {
        log.Fatalf("Failed to chat: %v", err)
    }
    log.Printf("chat: %v", chatResp)
}

func chat(prompt string) (string, error) {
    // URL
    url := "http://host.docker.internal:11434/api/chat"

    // リクエストボディを作成
    jsonData := []byte(fmt.Sprintf(`{
        "model": "llm",
        "stream": false,
        "messages": [
            { "role": "user", "content": "%v" }
        ]
    }`, prompt))

    // POSTリクエストを作成
    req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
    if err != nil {
        return "", fmt.Errorf("Failed to create request: %v", err)
    }
    req.Header.Set("Content-Type", "application/json")

    // クライアントを作成
    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        return "", fmt.Errorf("Failed to make POST request: %v", err)
    }
    defer resp.Body.Close()

    // レスポンスボディを読み取る
    body, err := ioutil.ReadAll(resp.Body)
    if err != nil {
        return "", fmt.Errorf("Failed to read response body: %v", err)
    }

    // レスポンスボディをパース
    var chatResp ChatResponse
    err = json.Unmarshal(body, &chatResp)
    if err != nil {
        log.Printf("Body: %v", string(body))
        return "", fmt.Errorf("Failed to parse response body: %v", err)
    }

    // レスポンス
    log.Printf("resp: %v", string(body))
    chat := chatResp.Message.Content
    return chat, nil
}

func embeddingText(prompt string) ([]float32, error){
    // URL
    url := "http://host.docker.internal:11434/api/embeddings"

    // リクエストボディを作成
    jsonData := []byte(fmt.Sprintf(`{
        "model": "llm",
        "prompt": "%v"
    }`, prompt))

    // POSTリクエストを作成
    req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
    if err != nil {
        return nil, fmt.Errorf("Failed to create request: %v", err)
    }
    req.Header.Set("Content-Type", "application/json")

    // クライアントを作成
    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        return nil, fmt.Errorf("Failed to make POST request: %v", err)
    }
    defer resp.Body.Close()

    // レスポンスボディを読み取る
    body, err := ioutil.ReadAll(resp.Body)
    if err != nil {
        return nil, fmt.Errorf("Failed to read response body: %v", err)
    }

    // レスポンスボディをパース
    var embeddingResp EmbeddingResponse
    err = json.Unmarshal(body, &embeddingResp)
    if err != nil {
        log.Printf("Body: %v", string(body))
        return nil, fmt.Errorf("Failed to parse response body: %v", err)
    }

    // 埋め込みベクトルを取得
    embedding := embeddingResp.Embedding
    return embedding, nil
}

func Search(indexPath string) {
    text := "RAGという検索手法について教えてください。"
    embedding, err := embeddingText(text)
    if err != nil {
        log.Fatalf("%v", err)
    }

    // usearchでインデックスを作成
    vectorSize := len(embedding)
    conf := usearch.DefaultConfig(uint(vectorSize))
    index, err := usearch.NewIndex(conf)
    if err != nil {
        log.Fatalf("Failed to create Index: %v", err)
    }
    defer index.Destroy()

    // Search
    err = index.Load(indexPath)
    if err != nil {
        log.Fatalf("Failed to load Index: %v", err)
    }

    keys, distances, err := index.Search(embedding, 3)
    if err != nil {
        log.Fatalf("Failed to search")
    }
    for i:=0; i < len(keys); i++ {
        log.Printf("key: %v, distance: %v", keys[i], distances[i])
    }
}
