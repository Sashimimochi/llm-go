# LLM Go

Go言語だけで完結するLLMアプリを作る。

## QuickStart

```bash
# Launch App
$ make up
# Sample Query 1
$ curl "http://localhost:8080/chat?prompt=RAGという検索手法について簡潔に教えてください。"
{"message":"RAG（Reasoning from Abstraction to Ground）とは、LLM（大規模言語モデル）に対して、プロンプトと外部情報源の検索結果を組み合わせて回答をさせる検索手法です。プロンプトに含まれるリファレンス情報を基に、モデルはユーザーの要望に正確な回答を提供することができます。"}
# Sample Queryv2
$ curl "http://localhost:8080/chat?prompt=HNSWという検索手法について簡潔に教えてください。"
{"message":"HNSW(Hierarchical Navigable Small World)は、グラフ探索型の検索手法で、以下のような特徴があります。\n\n1. 事前にベクトル距離に応じてグラフを構築する。\n2. 探索は高速である。\n3. 事前に構築したグラフの読み込みに時間がかかる。\n4. メモリ上に生データを展開して処理するため、メモリ消費量が大きくなる。\n\n2018年時点では、million-scaleのドキュメントの近似近傍探索で決定版と言われていましたが、2020年にGoogleがScaNNというより高速で高精度・高効率なアルゴリズムを提唱しています。"}
```

## References

- https://github.com/unum-cloud/usearch/
- https://github.com/ollama/ollama/
- https://huggingface.co/elyza/Llama-3-ELYZA-JP-8B-GGUF
