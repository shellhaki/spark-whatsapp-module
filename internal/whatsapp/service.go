package whatsapp

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/lib/pq"
	"github.com/mdp/qrterminal/v3"
	"github.com/shellhaki/spark-whatsapp-module/internal/config"
	"github.com/shellhaki/spark-whatsapp-module/internal/messages"
	"github.com/shellhaki/spark-whatsapp-module/internal/subscribers"
	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
	"google.golang.org/protobuf/proto"
)

type Service struct {
	client          *whatsmeow.Client
	subscribers     *subscribers.Repository
	subscribeWord   string
	unsubscribeWord string
}

func NewService(ctx context.Context, cfg config.Config, db *sql.DB, repo *subscribers.Repository) (*Service, error) {
	sqlstore.PostgresArrayWrapper = pq.Array

	container := sqlstore.NewWithDB(db, "postgres", waLog.Noop)
	if err := container.Upgrade(ctx); err != nil {
		return nil, fmt.Errorf("upgrade whatsapp sql store: %w", err)
	}

	device, err := container.GetFirstDevice(ctx)
	if err != nil {
		return nil, fmt.Errorf("get whatsapp device: %w", err)
	}

	service := &Service{
		client:          whatsmeow.NewClient(device, waLog.Noop),
		subscribers:     repo,
		subscribeWord:   normalizeCommand(cfg.SubscribeWord),
		unsubscribeWord: normalizeCommand(cfg.UnsubscribeWord),
	}

	service.client.AddEventHandler(service.handleEvent)

	go service.connect(ctx)

	return service, nil
}

func (s *Service) Close() {
	if s.client != nil {
		s.client.Disconnect()
	}
}

func (s *Service) SendToSubscribers(ctx context.Context, request messages.DeliveryRequest) ([]messages.DeliveryResult, error) {
	items, err := s.subscribers.ListActive(ctx)
	if err != nil {
		return nil, err
	}

	results := make([]messages.DeliveryResult, 0, len(items))
	for _, item := range items {
		jid, err := types.ParseJID(item.JID)
		if err != nil {
			return nil, fmt.Errorf("parse subscriber jid %s: %w", item.JID, err)
		}

		resp, err := s.send(ctx, jid, request)
		if err != nil {
			return nil, fmt.Errorf("send to %s: %w", item.JID, err)
		}

		results = append(results, messages.DeliveryResult{
			JID:       item.JID,
			MessageID: resp.ID,
		})
	}

	return results, nil
}

func (s *Service) SendToPhoneNumber(ctx context.Context, request messages.DirectDeliveryRequest) (messages.DeliveryResult, error) {
	subscriber, err := s.subscribers.FindActiveByPhoneNumber(ctx, request.PhoneNumber)
	if err != nil {
		if errors.Is(err, subscribers.ErrSubscriberNotFound) {
			return messages.DeliveryResult{}, err
		}
		return messages.DeliveryResult{}, err
	}

	jid, err := types.ParseJID(subscriber.JID)
	if err != nil {
		return messages.DeliveryResult{}, fmt.Errorf("parse subscriber jid %s: %w", subscriber.JID, err)
	}

	resp, err := s.send(ctx, jid, messages.DeliveryRequest{
		Text: request.Text,
	})
	if err != nil {
		return messages.DeliveryResult{}, fmt.Errorf("send to %s: %w", subscriber.JID, err)
	}

	return messages.DeliveryResult{
		JID:       subscriber.JID,
		MessageID: resp.ID,
	}, nil
}

func (s *Service) handleEvent(raw any) {
	switch evt := raw.(type) {
	case *events.Message:
		s.handleIncomingMessage(evt)
	}
}

func (s *Service) handleIncomingMessage(evt *events.Message) {
	if evt.Info.IsFromMe || evt.Info.IsGroup {
		return
	}

	text := normalizeCommand(evt.Message.GetConversation())
	if text == "" && evt.Message.GetExtendedTextMessage() != nil {
		text = normalizeCommand(evt.Message.GetExtendedTextMessage().GetText())
	}

	if text == "" {
		return
	}

	fmt.Printf(
		"[WHATSAPP MESSAGE] text=%q source=%s chat=%s sender=%s sender_alt=%s recipient_alt=%s mode=%s\n",
		text,
		evt.Info.SourceString(),
		evt.Info.Chat.String(),
		evt.Info.Sender.String(),
		evt.Info.SenderAlt.String(),
		evt.Info.RecipientAlt.String(),
		evt.Info.AddressingMode,
	)

	replyJID := resolveReplyJID(evt.Info)
	subscriberJID := resolveSubscriberJID(evt.Info)
	if replyJID.IsEmpty() || subscriberJID.IsEmpty() {
		fmt.Printf(
			"[WHATSAPP ERROR] could not resolve sender for text=%q reply_jid=%s subscriber_jid=%s\n",
			text,
			replyJID.String(),
			subscriberJID.String(),
		)
		return
	}

	ctx := context.Background()

	switch text {
	case s.subscribeWord:
		fmt.Printf(
			"[SUBSCRIPTION] subscribe command received subscriber_jid=%s reply_jid=%s push_name=%q\n",
			subscriberJID.String(),
			replyJID.String(),
			evt.Info.PushName,
		)

		active, err := s.subscribers.IsActive(ctx, subscriberJID.String())
		if err != nil {
			fmt.Printf("[SUBSCRIPTION ERROR] active check failed jid=%s err=%v\n", subscriberJID.String(), err)
			s.replyText(ctx, replyJID, "Could not check subscription right now.")
			return
		}
		if active {
			fmt.Printf("[SUBSCRIPTION] already subscribed jid=%s\n", subscriberJID.String())
			s.replyText(ctx, replyJID, "Already subscribed!")
			return
		}

		fmt.Printf(
			"[SUBSCRIPTION] saving subscriber jid=%s phone_number=%s push_name=%q\n",
			subscriberJID.String(),
			subscriberJID.User,
			evt.Info.PushName,
		)
		if err := s.subscribers.Subscribe(ctx, subscriberJID.String(), subscriberJID.User, evt.Info.PushName); err != nil {
			fmt.Printf("[SUBSCRIPTION ERROR] save failed jid=%s err=%v\n", subscriberJID.String(), err)
			s.replyText(ctx, replyJID, "Could not save subscription right now.")
			return
		}
		fmt.Printf("[SUBSCRIPTION] subscribed jid=%s via=%s\n", subscriberJID.String(), replyJID.String())
		s.replyText(ctx, replyJID, "Subscribed!")
	case s.unsubscribeWord:
		fmt.Printf(
			"[SUBSCRIPTION] unsubscribe command received subscriber_jid=%s reply_jid=%s\n",
			subscriberJID.String(),
			replyJID.String(),
		)

		unsubscribed, err := s.subscribers.Unsubscribe(ctx, subscriberJID.String())
		if err != nil {
			fmt.Printf("[SUBSCRIPTION ERROR] unsubscribe failed jid=%s err=%v\n", subscriberJID.String(), err)
			s.replyText(ctx, replyJID, "Could not update subscription right now.")
			return
		}
		if !unsubscribed {
			fmt.Printf("[SUBSCRIPTION] already unsubscribed jid=%s\n", subscriberJID.String())
			s.replyText(ctx, replyJID, "Already unsubscribed!")
			return
		}
		fmt.Printf("[SUBSCRIPTION] unsubscribed jid=%s via=%s\n", subscriberJID.String(), replyJID.String())
		s.replyText(ctx, replyJID, "Unsubscribed!")
	}
}

func (s *Service) send(ctx context.Context, jid types.JID, request messages.DeliveryRequest) (whatsmeow.SendResponse, error) {
	message := &waProto.Message{
		Conversation: proto.String(request.Text),
	}
	return s.client.SendMessage(ctx, jid, message)
}

func (s *Service) replyText(ctx context.Context, jid types.JID, text string) {
	_, err := s.client.SendMessage(ctx, jid, &waProto.Message{
		Conversation: proto.String(text),
	})
	if err != nil {
		fmt.Printf("[WHATSAPP REPLY ERROR] to=%s text=%q err=%v\n", jid.String(), text, err)
		return
	}
	fmt.Printf("[WHATSAPP REPLY] to=%s text=%q\n", jid.String(), text)
}

func (s *Service) connect(ctx context.Context) {
	if s.client.Store.ID == nil {
		qrChan, err := s.client.GetQRChannel(ctx)
		if err != nil {
			fmt.Printf("[WHATSAPP ERROR] get qr channel: %v\n", err)
			return
		}

		if err := s.client.Connect(); err != nil {
			fmt.Printf("[WHATSAPP ERROR] connect whatsapp client: %v\n", err)
			return
		}

		fmt.Println("Scan the QR code below to connect WhatsApp:")
		for evt := range qrChan {
			if evt.Event == "code" {
				qrterminal.GenerateHalfBlock(evt.Code, qrterminal.L, os.Stdout)
				continue
			}
			if evt.Event == "success" {
				fmt.Println("WhatsApp connected.")
			}
		}
		return
	}

	if err := s.client.Connect(); err != nil {
		fmt.Printf("[WHATSAPP ERROR] connect whatsapp client: %v\n", err)
	}
}

func normalizeCommand(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func resolveReplyJID(info types.MessageInfo) types.JID {
	if !info.Chat.IsEmpty() {
		return info.Chat.ToNonAD()
	}
	if !info.Sender.IsEmpty() {
		return info.Sender.ToNonAD()
	}
	return types.EmptyJID
}

func resolveSubscriberJID(info types.MessageInfo) types.JID {
	if jid := preferPhoneNumberJID(info.Sender, info.SenderAlt); !jid.IsEmpty() {
		return jid
	}
	if jid := preferPhoneNumberJID(info.Chat, info.RecipientAlt); !jid.IsEmpty() {
		return jid
	}
	return types.EmptyJID
}

func preferPhoneNumberJID(primary, alternate types.JID) types.JID {
	primary = primary.ToNonAD()
	alternate = alternate.ToNonAD()

	switch {
	case primary.Server == types.DefaultUserServer || primary.Server == types.LegacyUserServer:
		return primary
	case alternate.Server == types.DefaultUserServer || alternate.Server == types.LegacyUserServer:
		return alternate
	case !primary.IsEmpty():
		return primary
	case !alternate.IsEmpty():
		return alternate
	default:
		return types.EmptyJID
	}
}
