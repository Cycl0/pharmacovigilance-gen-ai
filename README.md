## Objective
Use llm models to analyze social media posts through the lenses of pharmacovigilance to verify if the user is talking about side effects of determined medicine and catalog this data.

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
