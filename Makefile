setup:
	wget --no-clobber https://huggingface.co/elyza/Llama-3-ELYZA-JP-8B-GGUF/resolve/main/Llama-3-ELYZA-JP-8B-q4_k_m.gguf?download=true -O ollama/elyza-jp8b.gguf || true

launch:
	docker-compose up -d
	docker-compose exec ollama ollama create llm -f Modelfile

up:
	@make setup
	@make launch

clean:
	docker-compose down --rmi all
	rm app/vector.index
	rm ollama/elyza-jp8b.gguf
