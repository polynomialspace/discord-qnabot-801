"Q&A" rating bot for a discord channel

WIP and will likely go unused


config.yaml format:
```
token: <your discord bot token here>
commandPrefix: "!sebot"
trackedReactions:
  Types:
    "❓": "question"
    "❔": "question"
    "❗": "answer"
    "❕": "answer"
  Ratings:
  -  "👍"
  -  "⬆️"
  -  "👎"
  -  "⬇️"
```