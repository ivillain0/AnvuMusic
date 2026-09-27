import requests
import os

response = requests.post(
    "https://codecraftapi.com/v1/chat/completions",
    headers={"Authorization": f"Bearer {os.environ['CODECRAFT_API_KEY']}"},
    json={
        "model": "claude-opus-5",
        "temperature": 1,
        "max_tokens": 8192,
        "messages": [
            {"role": "user", "content": "Hello!"}
        ]
    }
)
print(response.json()["choices"][0]["message"]["content"])
