package notification

import (
	"context"
	"log"
	"time"
	"fmt"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go/v4"
	"google.golang.org/api/option"
)

type NotificationService struct {
	firestoreClient *firestore.Client
}

func NewNotificationService() *NotificationService {
	// Inisialisasi Firebase Admin SDK
	ctx := context.Background()
	// Ganti "firebase-service-account.json" jika namamu berbeda
	opt := option.WithCredentialsFile("firebase-service-account.json")
	app, err := firebase.NewApp(ctx, nil, opt)
	if err != nil {
		log.Fatalf("Error initializing Firebase Admin SDK: %v\n", err)
	}

	// Dapatkan klien Firestore
	client, err := app.Firestore(ctx)
	if err != nil {
		log.Fatalf("Error initializing Firestore client: %v\n", err)
	}

	log.Println("Firebase Admin SDK and Firestore client initialized successfully.")
	return &NotificationService{firestoreClient: client}
}

// SendNotification mengirim notifikasi ke koleksi user di Firestore
func (s *NotificationService) SendNotification(userID int, title string, body string, notifType string) error {
	ctx := context.Background()

	// Definisikan data notifikasi
	notificationData := map[string]interface{}{
		"title":     title,
		"body":      body,
		"type":      notifType, // Misal: "deposit_success", "transfer_received"
		"is_read":   false,
		"timestamp": time.Now(),
	}

	// Tentukan koleksi: /notifications/{userID}/items
	// Kita gunakan ID user sebagai nama dokumen agar mudah di-query
	// Koreksi: Gunakan ID user sebagai NAMA KOLEKSI
	collectionPath := fmt.Sprintf("notifications_user_%d", userID)
    // Add() akan membuat dokumen dengan ID acak
	_, _, err := s.firestoreClient.Collection(collectionPath).Add(ctx, notificationData)

	if err != nil {
		log.Printf("Error sending notification to Firestore for user %d: %v", userID, err)
		return err
	}

	log.Printf("Successfully sent notification to user %d: %s", userID, title)
	return nil
}