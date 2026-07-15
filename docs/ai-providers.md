# Decision

Use an AI provider interface.


Example:


type Provider interface {

Generate(prompt string)(string,error)

}



Benefits:

Supports:

- OpenAI
- Anthropic
- Gemini
- Ollama
- OpenRouter


without changing core logic.