## Objective
- Use LLM models to analyze social media posts through the lens of pharmacovigilance.
- Find posts of users mentioning specific medicines.
- Infer if the posts are relevant, there is mention of any side effects, if identified, infer and extract this data.
- Finally catalog the data.

## Requirements
- Docker
- Python (Optional)
- Go (Optional)

## Environmental variables
- Create a .env file inside go/ (more up to date) or python/
- Replace it with your own respective api key, <username> and <password>
```sh
BLUESKY_USERNAME="<username>.bsky.social"
BLUESKY_APP_PASSWORD="xxxx-xxxx-xxxx-xxxx"
DEEPSEEK_API_KEY="sk-xxxxxxxxxxxxxxxxxxxxxxx"
OPENAI_API_KEY="sk-xxxxxxxxxxxxxxxxxxxxxxx"
OPENROUTER_API_KEY="sk-xxxxxxxxxxxxxxxxxxxxxxx"
MONGODB_URI="mongodb+srv://<username>:<password>@cluster0.mongodb.net/bluesky_data?retryWrites=true&w=majority"
```

## MongoDB Setup
- Create `bluesky_data` database and `posts` collection

## Build (Docker)
- Inside go/ (more up to date) or python/
``` sh
docker build -t get_posts .
```

## Run (Docker)
- `--network=host` is for local llm connection
``` sh
docker -run --network=host --rm get_posts
```

## Test Benchmarks with modern LLMs (as of March, 2025)
- The column `Detected` does not mean correctly detected, but the number of posts detected as relevant and had data extracted

| Medicine                        | Model Name                                                      | Posts Analyzed | Detected | Cost       | Time               |
|---------------------------------|-----------------------------------------------------------------|----------------|----------|------------|--------------------|
| Fluoxetina (Fluoxetine/Prozac)  | deepseek-chat (using prompt 1)                                  | 1043           | 89       | $US 0.05   | ~3h                |
| Fluoxetina (Fluoxetine/Prozac)  | deepseek-chat                                                   | 1043           | 203      | $US 0.03   | ~3h                |
| Fluoxetina (Fluoxetine/Prozac)  | chatgpt4o-latest                                                | 1043           | 151      | $US 1.20   | 6m16.567483523s    |
| Fluoxetina (Fluoxetine/Prozac)  | gpt-4o-mini                                                     | 1043           | 118      | $US 0.05   | 7m17.513542008s    |
| Fluoxetina (Fluoxetine/Prozac)  | gpt-4o-mini (second try)                                        | 1043           | 118      | $US 0.03   | 7m42.488382209s    |
| Fluoxetina (Fluoxetine/Prozac)  | local-FuseO1-DeepSeekR1-QwQ-Sky-32B-Q4_K_M                      | 491            | 93       | Free       | ~2h                |
| Fluoxetina (Fluoxetine/Prozac)  | local-IQ-quant-1i-FuseO1-DeepSeekR1-QwQ-Sky-32B-Q4_K_M          | 587            | 113      | Free       | ~2h                |
| Fluoxetina (Fluoxetine/Prozac)  | local-IQ-quant-1i-FuseO1-DeepSeekR1-QwQ-Sky-32B-Q4_K_M (second try) | 587       | 116      | Free       | ~2h                |
| Fluoxetina (Fluoxetine/Prozac)  | gemini-2.0-flash-001                                            | 1043           | 216      | Free       | 29m38.312395001s   |
| Fluoxetina (Fluoxetine/Prozac)  | qwen-max                                                        | 1043           | 120      | $US 0.544  | 14m31.540393825s   |
| Fluoxetina (Fluoxetine/Prozac)  | claude-3.7-sonnet                                               | 1043           | 151      | $US 1.169  | 30m57.261575826s   |

## Test Results
### Prompt 1 (only used for the first test with deepseek-chat)
- Answer with the side effects in english for yes and X for no.
				DO NOT EXPLAIN OR COMMENT
				The answer MUST above all be a single character or a list of just the name of the side effect without any aditional commentary/information/detail or an X for when this is not applied.
				Does this text talk about %s and its side effects.
				If the text only talks about %s and the symptoms that afflicts them but aren't specifically side effects from %s, answer with no (X).
				Answer with the main side effects in english only if the side effects are from %s and they are bad.
        If there are multiple side effects, separate them with a single comma without any whitespace
        Text: %s
### Prompt 2
- You are a pharmacovigilance specialist and you are analyzing the side effects regarding %s in social media posts.
        Answer with the side effects in english if there are any and X for no.
        YOU MUST BE ABLE TO UNDERSTAND AND INTERPRET INFORMAL LANGUAGE IN ANY LANGUAGE, YOU MUST NOT CONFUSE SIDE EFFECTS WITH THE SYMPTHOMS THE MEDICINE SOLVES OR GIVES WHEN ONE STOPS TAKING IT
        YOU MUST NOT ASSUME THE WHAT THE SIDE EFFECTS ARE, YOU SHOULD EXTRACT IT FROM THE TEXT
				DO NOT EXPLAIN OR COMMENT
				The answer MUST above all be a single character or a list of just the name of the side effect without any aditional commentary/information/detail or an X for when this is not applied.
				Does this post talk about %s and its side effects, physical or emotional?
				If the text only talks about %s and the symptoms that afflicts them but aren't specifically side effects from %s, answer with no (X).
				Answer with the main side effects in english only if the side effects are from %s and they are bad or undesirable.
        If there are multiple side effects, separate them with a single comma without any whitespace
        Post: %s
