# Retro: JWT Algorithm — 2026-08-07

## Mistake

The initial implementation used HMAC-SHA256 (HS256) for JWT signing. This
is symmetric — the same secret signs and verifies tokens. Every service
that needs to verify a token must have the secret.

## Root Cause

No rule explicitly required asymmetric JWT algorithms. The go-chi skill
doesn't mention JWT algorithm choice. DeepSeek defaulted to HMAC because
it's simpler to implement (no key generation, no key files).

## Rule Updated

**File:** `.agents/skills/go-chi/SKILL.md`

Added under "Error Handling" → new "Authentication" section:

```
### DO — Use asymmetric JWT (ES256/RSA256) not HMAC

- Generate key pair: `openssl ecparam -genkey -name prime256v1 -noout -out private.pem`
- Extract public key: `openssl ec -in private.pem -pubout -out public.pem`
- Load keys at startup from env vars or files
- Sign with private key, verify with public key
- HMAC (HS256) only acceptable for single-service prototypes
```

**Files changed:**
- `backend/cmd/server/main.go` — load ECDSA key pair from env/pem files
- `backend/internal/modules/auth/application/service.go` — sign with ECDSA private key
- `backend/internal/modules/auth/adapter/http/handler.go` — verify with ECDSA public key
- `.agents/skills/go-chi/SKILL.md` — added JWT algorithm guidance

## Transcript

User: "I am checking that we are not using assymetric encryption for jwt,
can we change that?"

The agent had implemented HS256 because it was the simplest path and no
rule said otherwise. The correct pattern for a service that may eventually
have multiple consumers (frontend, mobile, potential microservices) is
asymmetric keys.
