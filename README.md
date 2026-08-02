# llm

A learning project: building LLM chat, tool-calling agents, and structured output in Go, from scratch and with an SDK, side by side.

Every endpoint is served by **two interchangeable implementations** of the same interface:

| | package | how it talks to Groq |
|---|---|---|
| **raw** | `internal/service/groqrawapi` | hand-rolled `net/http` against the chat completions API |
| **lib** | `internal/service/groqlib` | the community SDK [`conneroisu/groq-go`](https://github.com/conneroisu/groq-go) |

The backend is [Groq](https://groq.com) (OpenAI-compatible), default model `openai/gpt-oss-120b`.

## Running it

Requires Go 1.24+ and a Groq API key ([free tier is enough](https://console.groq.com)).

```bash
cp .env.example .env
go run .
```

| variable | default | |
|---|---|---|
| `GROQ_API_KEY` | — | **required** |
| `GROQ_MODEL_NAME` | `openai/gpt-oss-120b` | |
| `HTTP_ADDR` | `:8080` | |

## Endpoints

The four chat endpoints take the same body and return the same shape.

```jsonc
{
  "content": "message",             // required
  "session_id": "uuid",             // optional; omit to start a new conversation
  "implementation": "groq_raw_api"  // optional; anything else selects the SDK impl
}
```

| method | endpoint | what it does |
|---|---|---|
| `POST` | `/chat/no-memory` | one stateless call, no history |
| `POST` | `/chat` | multi-turn, history kept per `session_id` |
| `POST` | `/chat/agent/weather` | tool-calling agent, one tool |
| `POST` | `/chat/agent/log-triage` | tool-calling agent, three tools |
| `POST` | `/study-plan` | structured output — see [below](#structured-output), different body |

### A plain chat turn

```bash
curl -s localhost:8080/chat -d '{"content":"explain goroutines in one sentence"}'
```

Pass the returned `session_id` back to continue the conversation:

```bash
curl -s localhost:8080/chat -d '{"session_id":"<id>","content":"now compare them to OS threads"}'
```

### The weather agent

```bash
curl -s localhost:8080/chat/agent/weather -d '{"content":"is it jacket weather in Medan?"}'
```

One tool, `get_weather`, returning random data. It exists to make the tool loop observable end to end — the model asks for the tool, the server runs it, feeds the result back, and the model answers.

### The log-triage agent

Three tools (`query_logs`, `get_metrics`, `search_runbook`) over a small fake incident:

> `checkout` p99 spiked at 14:03. But checkout is fine — its DB pool is healthy and traffic is normal. It calls `payment`, whose `max_pool_size` was quietly lowered 20 → 5 by a config reload at **13:58**. Payment saturated its pool and started timing out. Checkout is the victim, not the cause.

```bash
curl -s localhost:8080/chat/agent/log-triage \
  -d '{"content":"what happened to checkout service at around 14:00?"}'
```

Getting this right takes about six tool calls: read checkout logs → rule out its DB → rule out a traffic spike → hop to payment → confirm the saturated pool → find the runbook. Watch the server log to see which path the model actually took.

```
2026/07/26 08:04:46 on iteration 7, call tool search_runbook with args: {"query":"pool exhausted"}
2026/07/26 08:04:49 usage: session=c5fd2610-6f18-4737-953f-53f585db4d75 iteration=8 prompt_tokens=1158 completion_tokens=995
```

Every model call logs its token usage with the session and iteration, so you can watch context grow across a conversation — the numbers climb fast once tool results start accumulating.

### Structured output

`/study-plan` asks the model to fill in a Go struct rather than reply in prose. It takes a different body — no `content`, no session:

```bash
curl -s localhost:8080/study-plan -d '{
  "param": {"topic":"go concurrency","current_level":"beginner","weeks":4,"hours_per_week":6}
}'
```

```jsonc
{
  "implementation": "groq_raw_api",  // optional, as everywhere else
  "param": {
    "topic": "go concurrency",
    "current_level": "beginner",     // beginner | intermediate | advanced
    "weeks": 4,
    "hours_per_week": 6
  }
}
```

The reply is a `domain.StudyPlan`: a topic, a total, a list of weeks (each with focus, activities, and an outcome), and the assumptions the model made.

## Layout

```
main.go                       server, graceful shutdown
internal/config               env loading
internal/handler              chi routes, request decoding, impl selection
internal/service              the Service interface both impls satisfy
internal/service/groqrawapi   raw net/http implementation
internal/service/groqlib      groq-go SDK implementation
internal/domain               shared request/response types for structured output
internal/mocktool             fake tools: weather + the log-triage world
```

## Known limitations

Deliberate, given the scope, but real:

- **Sessions are per-implementation.** The two services own separate stores, so reusing a `session_id` while switching `implementation` silently starts a fresh history under the same ID.
- **The system prompt is only set on a session's first turn**, so continuing a `/chat` session on an agent endpoint keeps the original prompt while swapping in the agent's tools.
- **No iteration cap on the agent loop** yet.
- **No tests.** The session, history, and schema-generation logic is pure and would be easy to cover; it just isn't yet.
- The two implementations send slightly different JSON schemas for `/study-plan` (see above). Both are valid, but it means structured output is not byte-for-byte identical across them.
- The raw implementation's Groq URL is hardcoded, so it can't be pointed at a test server the way the SDK client can.
- Every upstream failure surfaces as `500`, losing the distinction between a rate limit, a timeout, and a bug.
- Session expiry is measured from creation, not last use, so a long conversation is evicted 15 minutes in.
- In-memory only, no auth, single user. Not built to deploy.

## License

[MIT](LICENSE)
