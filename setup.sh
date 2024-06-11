#!/bin/bash

sudo docker-compose up -d
wget --no-clobber https://huggingface.co/mmnga/alfredplpl-Llama-3-8B-Instruct-Ja-gguf/resolve/main/alfredplpl-Llama-3-8B-Instruct-Ja-Q5_K_M.gguf?download=true -O ollama/model.gguf
docker-compose exec ollama ollama create llm -f Modelfile
