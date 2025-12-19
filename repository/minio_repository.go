package repository

import (
	"bytes"
	"context"
	"encoding/json"
	"go-microservice/models"
	"go-microservice/utils"
	"io"
	"sync"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type MinIoRepository struct {
	log        *utils.Logger
	client     *minio.Client
	bucket     string
	filename   string
	users      []models.User
	ctxTimeout int
	mu         sync.RWMutex
}

func NewMinIoRepository(endpoint, username, password, bucket, filename string, ctx context.Context, ctxTimeout int) *MinIoRepository {
	log := utils.GlobalLogger()

	log.Debug("Try to create MinIO connection...")
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(username, password, ""),
		Secure: false, // use SSL false hardcoded
	})

	if err != nil {
		log.Critical("No minIO connection: %+v", err)
		return nil
	}

	log.Info("MinIO connection established!")

	connCtx, cancel := context.WithTimeout(ctx, time.Duration(ctxTimeout)*time.Second)
	defer cancel()

	result := MinIoRepository{
		log:        log,
		client:     client,
		bucket:     bucket,
		filename:   filename,
		ctxTimeout: ctxTimeout,
	}

	exists, _ := client.BucketExists(connCtx, bucket)
	if !exists {
		log.Info("No bucket. Create it")
		client.MakeBucket(connCtx, bucket, minio.MakeBucketOptions{})
	}

	if !result.loadUsers(ctx) {
		err := result.saveUsers(ctx)
		if err != nil {
			log.Critical("Cann't create users file: %+v", err)
			return nil
		}
	}

	return &result
}

func (r *MinIoRepository) loadUsers(ctx context.Context) bool {
	connCtx, cancel := context.WithTimeout(ctx, time.Duration(r.ctxTimeout)*time.Second)
	defer cancel()
	obj, err := r.client.GetObject(connCtx, r.bucket, r.filename, minio.GetObjectOptions{})
	if err != nil {
		r.log.Error("Cann't read file: %+v", err)
		return false
	}
	defer obj.Close()

	data, err := io.ReadAll(obj)
	if err != nil {
		r.log.Error("Read file error: %+v", err)
		r.removeAllUsers(ctx)
		return false
	}

	if len(data) == 0 {
		r.log.Info("File is empty")
		return false
	}

	if err := json.Unmarshal(data, &r.users); err != nil {
		r.log.Error("File is not json: %+v", err)
		r.removeAllUsers(ctx)
		return false
	}

	r.log.Info("All Users load OK!")

	return true
}

func (r *MinIoRepository) saveUsers(ctx context.Context) error {
	data, _ := json.MarshalIndent(r.users, "", "  ")
	reader := bytes.NewReader(data)

	connCtx, cancel := context.WithTimeout(ctx, time.Duration(r.ctxTimeout)*time.Second)
	defer cancel()
	_, err := r.client.PutObject(
		connCtx,
		r.bucket,
		r.filename,
		reader,
		int64(len(data)),
		minio.PutObjectOptions{ContentType: "application/json"},
	)
	return err
}

func (r *MinIoRepository) removeAllUsers(ctx context.Context) {

	connCtx, cancel := context.WithTimeout(ctx, time.Duration(r.ctxTimeout)*time.Second)
	defer cancel()

	err := r.client.RemoveObject(connCtx, r.bucket, r.filename, minio.RemoveObjectOptions{})
	if err != nil {
		r.log.Critical("Cann't delete users: %+v", err)
		return
	}
	r.users = make([]models.User, 0)
	r.log.Info("All Users deleted!")
}

func (r *MinIoRepository) Close() {
	r.log.Info("Stop MinIO context!")
}

func (r *MinIoRepository) GetAllUsers() []models.User {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.users
}

func (r *MinIoRepository) GetUserById(id int) *models.User {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for i := range r.users {
		if r.users[i].ID == id {
			return &r.users[i]
		}
	}
	return nil
}

func (r *MinIoRepository) checkForUserExists(name, email string) bool {
	for _, user := range r.users {
		if user.Name == name || user.Email == email {
			return true
		}
	}
	return false
}

func (r *MinIoRepository) AddNewUser(name, email string, ctx context.Context) *models.User {
	r.mu.Lock()
	defer r.mu.Unlock()

	check := r.checkForUserExists(name, email)
	if check {
		return nil
	}
	user := models.User{
		ID:    len(r.users),
		Name:  name,
		Email: email,
	}

	r.users = append(r.users, user)
	err := r.saveUsers(ctx)
	if err != nil {
		r.log.Critical("Save users error: %+v", err)
		return nil
	}
	return &user
}

func (r *MinIoRepository) Update(ctx context.Context) error {
	return r.saveUsers(ctx)
}
