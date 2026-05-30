package subscribers

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode"
)

var ErrSubscriberNotFound = errors.New("subscriber not found")

type Subscriber struct {
	JID            string     `json:"jid"`
	PhoneNumber    string     `json:"phone_number"`
	PushName       string     `json:"push_name"`
	Subscribed     bool       `json:"subscribed"`
	SubscribedAt   time.Time  `json:"subscribed_at"`
	UnsubscribedAt *time.Time `json:"unsubscribed_at,omitempty"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Subscribe(ctx context.Context, jid, phoneNumber, pushName string) error {
	phoneNumber = NormalizePhoneNumber(phoneNumber)

	query := `
INSERT INTO whatsapp_subscribers (
    jid,
    phone_number,
    push_name,
    subscribed,
    subscribed_at,
    unsubscribed_at,
    updated_at
) VALUES ($1, $2, $3, TRUE, NOW(), NULL, NOW())
ON CONFLICT (jid) DO UPDATE
SET phone_number = EXCLUDED.phone_number,
    push_name = EXCLUDED.push_name,
    subscribed = TRUE,
    subscribed_at = NOW(),
    unsubscribed_at = NULL,
    updated_at = NOW()
`

	if _, err := r.db.ExecContext(ctx, query, jid, phoneNumber, pushName); err != nil {
		return fmt.Errorf("subscribe jid %s: %w", jid, err)
	}

	return nil
}

func (r *Repository) IsActive(ctx context.Context, jid string) (bool, error) {
	row := r.db.QueryRowContext(ctx, `
SELECT subscribed
FROM whatsapp_subscribers
WHERE jid = $1
LIMIT 1
`, jid)

	var subscribed bool
	if err := row.Scan(&subscribed); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, fmt.Errorf("check subscriber %s: %w", jid, err)
	}

	return subscribed, nil
}

func (r *Repository) Unsubscribe(ctx context.Context, jid string) (bool, error) {
	query := `
UPDATE whatsapp_subscribers
SET subscribed = FALSE,
    unsubscribed_at = NOW(),
    updated_at = NOW()
WHERE jid = $1 AND subscribed = TRUE
`

	result, err := r.db.ExecContext(ctx, query, jid)
	if err != nil {
		return false, fmt.Errorf("unsubscribe jid %s: %w", jid, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("unsubscribe jid %s rows affected: %w", jid, err)
	}

	return rowsAffected > 0, nil
}

func (r *Repository) ListActive(ctx context.Context) ([]Subscriber, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT jid, phone_number, push_name, subscribed, subscribed_at, unsubscribed_at, updated_at
FROM whatsapp_subscribers
WHERE subscribed = TRUE
ORDER BY subscribed_at ASC
`)
	if err != nil {
		return nil, fmt.Errorf("list active subscribers: %w", err)
	}
	defer rows.Close()

	var result []Subscriber
	for rows.Next() {
		var item Subscriber
		if err := rows.Scan(
			&item.JID,
			&item.PhoneNumber,
			&item.PushName,
			&item.Subscribed,
			&item.SubscribedAt,
			&item.UnsubscribedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan subscriber: %w", err)
		}
		result = append(result, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate subscribers: %w", err)
	}

	return result, nil
}

func (r *Repository) FindActiveByPhoneNumber(ctx context.Context, phoneNumber string) (Subscriber, error) {
	phoneNumber = NormalizePhoneNumber(phoneNumber)

	row := r.db.QueryRowContext(ctx, `
SELECT jid, phone_number, push_name, subscribed, subscribed_at, unsubscribed_at, updated_at
FROM whatsapp_subscribers
WHERE phone_number = $1 AND subscribed = TRUE
LIMIT 1
`, phoneNumber)

	var item Subscriber
	if err := row.Scan(
		&item.JID,
		&item.PhoneNumber,
		&item.PushName,
		&item.Subscribed,
		&item.SubscribedAt,
		&item.UnsubscribedAt,
		&item.UpdatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Subscriber{}, ErrSubscriberNotFound
		}
		return Subscriber{}, fmt.Errorf("find subscriber by phone number %s: %w", phoneNumber, err)
	}

	return item, nil
}

func NormalizePhoneNumber(value string) string {
	value = strings.TrimSpace(value)

	var builder strings.Builder
	builder.Grow(len(value))
	for _, char := range value {
		if unicode.IsDigit(char) {
			builder.WriteRune(char)
		}
	}

	return builder.String()
}
