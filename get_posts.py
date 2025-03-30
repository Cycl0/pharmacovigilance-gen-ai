#!/usr/bin/env python3

import os
import requests
import time
from datetime import datetime, timezone
from dateutil.parser import isoparse
from pymongo import MongoClient, errors
from dotenv import load_dotenv

# Credenciais
load_dotenv()
USERNAME = os.getenv("BLUESKY_USERNAME")
APP_PASSWORD = os.getenv("BLUESKY_APP_PASSWORD")
API_KEY = os.getenv("DEEPSEEK_API_KEY")

# MongoDB Config
client = MongoClient(os.getenv("MONGODB_URI"))
db = client.bluesky_data
db.posts.create_index("post_uri", unique=True)

## DeepSeek
API_URL = "https://api.deepseek.com/v1/chat/completions"

headers = {
    "Authorization": f"Bearer {API_KEY}",
    "Content-Type": "application/json"
}


searchTerm = "Fluoxetina"


def generate_text(prompt: str, model: str = "deepseek-chat", max_tokens: int = 500):
    data = {
        "model": model,
        "messages": [{"role": "user", "content": prompt}],
        "temperature": 0.7,
        "max_tokens": max_tokens,
    }

    try:
        response = requests.post(API_URL, headers=headers, json=data)
        response.raise_for_status()  # Raise HTTP errors
        result = response.json()
        return result["choices"][0]["message"]["content"]
    except requests.exceptions.RequestException as e:
        print(f"API Request Failed: {e}")
        return None


## BlueSky
if not all([USERNAME, APP_PASSWORD]):
    raise ValueError("Missing credentials in .env file")

def authenticate():
    response = requests.post(
        "https://bsky.social/xrpc/com.atproto.server.createSession",
        json={"identifier": USERNAME, "password": APP_PASSWORD}
    )
    response.raise_for_status()
    return response.json()

auth_data = authenticate()
ACCESS_TOKEN = auth_data["accessJwt"]

# Iniciar sessao
session = requests.Session()
session.headers.update({"Authorization": f"Bearer {ACCESS_TOKEN}"})

def transform_post(post):
    """Converter post para documento no MongoDB"""
    return {
        "post_uri": post["uri"],
        "author": {
            "did": post["author"]["did"],
            "handle": post["author"]["handle"],
            "display_name": post["author"].get("displayName", "")
        },
        "content": post["record"]["text"],
        "created_at": isoparse(post["record"]["createdAt"]),
        "indexed_at": datetime.now(timezone.utc),
        "search_query": searchTerm
    }

def save_to_mongodb(posts):
    """Inserir em batch posts transformados"""
    if not posts:
        return 0

    transformed = [transform_post(p) for p in posts]

    try:
        result = db.posts.insert_many(transformed, ordered=False)
        return len(result.inserted_ids)
    except errors.BulkWriteError as e:
        print(f"Duplicatas pulados count: {len(e.details['writeErrors'])}")
        return len(transformed) - len(e.details['writeErrors'])


# Buscar posts
def get_all_posts(query: str, max_results: int = 1000):
    cursor = None
    total_retrieved = 0

    while True:
        params = {
            "q": query,
            "limit": 100 if max_results > 100 else max_results,
            "cursor": cursor
        }

        try:
            response = session.get(
                "https://bsky.social/xrpc/app.bsky.feed.searchPosts",
                params=params,
                timeout=10
            )
            response.raise_for_status()
            data = response.json()
        except requests.exceptions.HTTPError as e:
            if e.response.status_code == 429:
                print("Rate limit hit - add delay or stop")
                break
            else:
                raise

        posts = data.get("posts", [])
        relevant_posts=[]

        for post in posts:
            print(f"Autor: @{post['author']['handle']}")
            print(f"Texto: {post['record']['text']}")
            isRelevant = generate_text(f"""
            Answer with O for yes and X for no.
            DO NOT EXPLAIN OR COMMENT
            The answer MUST above all be a single character O or X.
            Does this text talk about {searchTerm} and its side effects.
            If the text only talks about {searchTerm} and the sympthoms that afflicts them but aren't specifically side effects from {searchTerm}, answer with no (X).
            Answer with yes (O) only if the side effects are from {searchTerm} and they are bad. Text: {post['record']['text']}""")
            print(f"Sobre efeitos colaterais: {isRelevant}\n---")
            if isRelevant == "O":
                relevant_posts.append(post)


        print("--------Posts relevantes--------")
        for post in relevant_posts:
            print({post['record']['text']})

        saved_count = save_to_mongodb(relevant_posts)

        total_retrieved += len(posts)

        time.sleep(1)
        print("Posts salvos: ", total_retrieved)
        cursor = data.get("cursor")
        if not cursor or total_retrieved >= max_results:
            break

get_all_posts(searchTerm, 10)
