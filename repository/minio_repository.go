package repository

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"go-microservice/models"
	"go-microservice/utils"
	"io"
	"sync"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type MinIoRepository struct {
	log          *utils.Logger
	client       *minio.Client
	bucket       string
	filename     string
	users        map[int]*models.UserMap
	ctxTimeout   int
	mu           sync.RWMutex
	maxCurrentId int
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

	result.initMaxId()

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
	r.users = make(map[int]*models.UserMap)
	r.log.Info("All Users deleted!")
}

func (r *MinIoRepository) initMaxId() {
	r.maxCurrentId = 0

	if len(r.users) == 0 {
		return
	}

	for id := range r.users {
		if id > r.maxCurrentId {
			r.maxCurrentId = id
		}
	}
	r.maxCurrentId++
}

func (r *MinIoRepository) Close() {
	r.log.Info("Stop MinIO context!")
}

func (r *MinIoRepository) GetAllUsers() []models.User {
	r.mu.RLock()
	defer r.mu.RUnlock()

	users := make([]models.User, 0)
	for id, user := range r.users {
		users = append(users, models.MapToUser(id, user))
	}

	return users
}

func (r *MinIoRepository) GetUserById(id int) *models.UserMap {
	r.mu.RLock()
	defer r.mu.RUnlock()

	user, exists := r.users[id]
	if !exists {
		return nil
	}

	return user
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

	id := r.maxCurrentId
	user := models.UserMap{Name: name, Email: email}
	r.users[id] = &user
	r.maxCurrentId++
	err := r.saveUsers(ctx)
	if err != nil {
		r.log.Critical("Save users error: %+v", err)
		return nil
	}
	result := models.MapToUser(id, &user)
	return &result
}

func (r *MinIoRepository) DeleteById(id int, ctx context.Context) error {
	_, exists := r.users[id]
	if !exists {
		return fmt.Errorf("user not found")
	}

	delete(r.users, id)
	err := r.saveUsers(ctx)
	return err
}

func (r *MinIoRepository) Update(ctx context.Context) error {
	return r.saveUsers(ctx)
}
