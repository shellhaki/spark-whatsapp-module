# spark-whatsapp-module

Small WhatsApp notification API built with `whatsmeow`.

It currently supports:
- WhatsApp opt-in with `subscribe`
- WhatsApp opt-out with `unsubscribe`
- direct text sends to one subscribed number
- broadcast text sends to all subscribed users

For speed, the app uses:
- SQLite as the primary runtime database
- optional Postgres as async backup storage for subscriber writes

## How it works

1. A user sends `subscribe` to the bot on WhatsApp.
2. The bot replies and stores that phone number as subscribed.
3. Your backend can now:
   - send direct text to that subscribed number
   - send broadcast text to all subscribed numbers

If a number is not subscribed, direct send is rejected.

## API

### `POST /api/messages/direct`

Send text to one subscribed phone number.

Example body:

```json
{
  "phone_number": "2348000000000",
  "message": "hello from spark"
}
```

### `POST /api/messages/broadcast`

Send text to all subscribed users.

Example body:

```json
{
  "message": "hello subscribers"
}
```

## Configuration

Copy `.env.example` to `.env` and edit it:

```bash
cp .env.example .env
```

Available environment variables:

```env
SQLITE_PATH=spark-whatsapp-module.db
POSTGRES_URI=
HTTP_ADDRESS=:8080
WHATSAPP_SUBSCRIBE_WORD=subscribe
WHATSAPP_UNSUBSCRIBE_WORD=unsubscribe
DB_MAX_OPEN_CONNS=10
DB_MAX_IDLE_CONNS=5
DB_CONN_MAX_IDLE_TIME=5m
DB_CONN_MAX_LIFETIME=30m
```

Notes:
- `SQLITE_PATH` is required and is used by the main app flow.
- `POSTGRES_URI` is optional. If set, subscriber writes are mirrored in the background.
- if Postgres is down, the app still keeps working on SQLite.

### Getting `POSTGRES_URI` from SparkDB

If you are using [SparkDB](https://sparkdb.pro/):

1. Sign in at `https://sparkdb.pro/auth`
2. Open `https://sparkdb.pro/dashboard`
3. Create a Postgres database
4. Open that database's details or SDK/access page
5. Copy the Postgres connection URL
6. Paste it into `.env` as:

```env
POSTGRES_URI=postgres://user:password@host:port/database
```

This project needs the Postgres connection URL, not the SparkDB API key.

## Local development

Initialize tables:

```bash
make setup
```

Run the app:

```bash
make run
```

You can also use:

```bash
go run ./cmd/setup
go run ./cmd/spark-whatsapp-module
```

## REST testing

Request files are in [requests/](/home/haki/spark/spark-whatsapp-module/requests):
- [requests/direct.http](/home/haki/spark/spark-whatsapp-module/requests/direct.http:1)
- [requests/broadcast.http](/home/haki/spark/spark-whatsapp-module/requests/broadcast.http:1)

## Manual deploy on a VPS

Install system packages:

```bash
sudo apt-get update
sudo apt-get install -y golang git build-essential sqlite3
```

Clone the project:

```bash
git clone <your-repo-url> spark-whatsapp-module
cd spark-whatsapp-module
```

Create config:

```bash
cp .env.example .env
```

Initialize the databases:

```bash
make setup
```

Build:

```bash
make build
```

Run manually:

```bash
./bin/app
```

## Docker deploy

Build:

```bash
docker build -t spark-whatsapp-module .
```

Run:

```bash
docker run -d \
  --name spark-whatsapp-module \
  --restart unless-stopped \
  -p 8080:8080 \
  -v $(pwd)/data:/app/data \
  --env-file .env \
  spark-whatsapp-module
```

Recommended for Docker:
- set `SQLITE_PATH=/app/data/spark-whatsapp-module.db`
- mount `/app/data` so the SQLite database survives container restarts

## systemd deploy

There is a first-install helper script:

```bash
sudo bash scripts/install-vps.sh
```

This is for first installation only.

It:
- installs required packages
- copies the project into `/opt/spark-whatsapp-module`
- creates `/etc/spark-whatsapp-module.env`
- builds the binary
- installs a `systemd` service
- enables and starts the service

After the first install, normal updates are:

```bash
git pull
sudo systemctl restart spark-whatsapp-module
```

Useful service commands:

```bash
sudo systemctl status spark-whatsapp-module
sudo journalctl -u spark-whatsapp-module -f
```

## Open source notes

This repo is set up to be minimally open-source friendly:
- runtime secrets stay in `.env`
- `.env.example` shows the required keys without secrets
- SQLite database files are local runtime data and should not be committed

Before publishing, make sure your real `.env`, database files, and WhatsApp session data are not in git.
