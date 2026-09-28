package mongo

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"time"

	"basic-app/models"
	"basic-app/repository"
	"basic-app/services"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	mongodriver "go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const databaseTimeout = 5 * time.Second

type UserRepository struct {
	collection *mongodriver.Collection
}

func NewUserRepository(
	db *mongodriver.Database,
) *UserRepository {
	return &UserRepository{
		collection: db.Collection("users"),
	}
}

// Create creates a new user.
func (r *UserRepository) Create(
	ctx context.Context,
	user *models.User,
) error {
	ctx, cancel := context.WithTimeout(ctx, databaseTimeout)
	defer cancel()

	now := time.Now().UTC()

	if user.ID.IsZero() {
		user.ID = bson.NewObjectID()
	}

	if user.CreatedAt.IsZero() {
		user.CreatedAt = now
	}

	user.UpdatedAt = now

	user.Email = strings.ToLower(
		strings.TrimSpace(user.Email),
	)

	user.Username = strings.TrimSpace(
		user.Username,
	)

	_, err := r.collection.InsertOne(
		ctx,
		user,
	)

	return err
}

// FindByID finds a user by MongoDB ObjectID.
func (r *UserRepository) FindByID(
	ctx context.Context,
	id string,
) (*models.User, error) {
	ctx, cancel := context.WithTimeout(ctx, databaseTimeout)
	defer cancel()

	objectID, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}

	var user models.User

	err = r.collection.FindOne(
		ctx,
		bson.M{
			"_id": objectID,
		},
	).Decode(&user)

	if err != nil {
		if errors.Is(err, mongodriver.ErrNoDocuments) {
			return nil, services.ErrUserNotFound
		}

		return nil, err
	}

	return &user, nil
}

// FindByUsername finds a user by username.
func (r *UserRepository) FindByUsername(
	ctx context.Context,
	username string,
) (*models.User, error) {
	ctx, cancel := context.WithTimeout(ctx, databaseTimeout)
	defer cancel()

	username = strings.TrimSpace(username)

	var user models.User

	err := r.collection.FindOne(
		ctx,
		bson.M{
			"username": username,
		},
	).Decode(&user)

	if err != nil {
		if errors.Is(err, mongodriver.ErrNoDocuments) {
			return nil, services.ErrUserNotFound
		}

		return nil, err
	}

	return &user, nil
}

// Update updates general user information.
func (r *UserRepository) Update(
	ctx context.Context,
	user *models.User,
) error {
	ctx, cancel := context.WithTimeout(ctx, databaseTimeout)
	defer cancel()

	user.UpdatedAt = time.Now().UTC()

	filter := bson.M{
		"_id": user.ID,
	}

	update := bson.M{
		"$set": bson.M{
			"first_name":    user.FirstName,
			"last_name":     user.LastName,
			"username":      strings.TrimSpace(user.Username),
			"email":         strings.ToLower(strings.TrimSpace(user.Email)),
			"alt_email":     user.AltEmail,
			"phone":         user.Phone,
			"date_of_birth": user.DateOfBirth,

			"address_line_1": user.AddressLine1,
			"address_line_2": user.AddressLine2,
			"city":           user.City,
			"state":          user.State,
			"pin_code":       user.PinCode,

			"updated_at": user.UpdatedAt,
		},
	}

	result, err := r.collection.UpdateOne(
		ctx,
		filter,
		update,
	)

	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return services.ErrUserNotFound
	}

	return nil
}

// UpdateProfilePic updates only the user's profile picture.
func (r *UserRepository) UpdateProfilePic(
	ctx context.Context,
	userID string,
	profilePic string,
) error {
	ctx, cancel := context.WithTimeout(ctx, databaseTimeout)
	defer cancel()

	objectID, err := bson.ObjectIDFromHex(userID)
	if err != nil {
		return err
	}

	update := bson.M{
		"$set": bson.M{
			"profile_pic": profilePic,
			"updated_at":  time.Now().UTC(),
		},
	}

	result, err := r.collection.UpdateOne(
		ctx,
		bson.M{
			"_id": objectID,
		},
		update,
	)

	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return services.ErrUserNotFound
	}

	return nil
}

// UpdatePassword updates the password hash.
func (r *UserRepository) UpdatePassword(
	ctx context.Context,
	userID string,
	passwordHash string,
) error {
	ctx, cancel := context.WithTimeout(ctx, databaseTimeout)
	defer cancel()

	objectID, err := bson.ObjectIDFromHex(userID)
	if err != nil {
		return err
	}

	update := bson.M{
		"$set": bson.M{
			"password_hash": passwordHash,
			"updated_at":    time.Now().UTC(),
		},
	}

	result, err := r.collection.UpdateOne(
		ctx,
		bson.M{
			"_id": objectID,
		},
		update,
	)

	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return services.ErrUserNotFound
	}

	return nil
}

// UpdateRefreshToken updates the stored refresh token hash.
func (r *UserRepository) UpdateRefreshToken(
	ctx context.Context,
	userID string,
	refreshTokenHash string,
) error {
	ctx, cancel := context.WithTimeout(ctx, databaseTimeout)
	defer cancel()

	objectID, err := bson.ObjectIDFromHex(userID)
	if err != nil {
		return err
	}

	now := time.Now().UTC()

	update := bson.M{
		"$set": bson.M{
			"refresh_token_hash": refreshTokenHash,
			"last_login_at":      now,
			"updated_at":         now,
		},
	}

	result, err := r.collection.UpdateOne(
		ctx,
		bson.M{
			"_id": objectID,
		},
		update,
	)

	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return services.ErrUserNotFound
	}

	return nil
}

// Change user account status without deleting it.
func (r *UserRepository) UserStatus(
	ctx context.Context,
	userID string,
	status bool,
) error {
	ctx, cancel := context.WithTimeout(ctx, databaseTimeout)
	defer cancel()

	objectID, err := bson.ObjectIDFromHex(userID)
	if err != nil {
		return err
	}

	update := bson.M{
		"$set": bson.M{
			"status":     status,
			"updated_at": time.Now().UTC(),
		},
	}

	result, err := r.collection.UpdateOne(
		ctx,
		bson.M{
			"_id": objectID,
		},
		update,
	)

	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return services.ErrUserNotFound
	}

	return nil
}

// Delete permanently removes a user from MongoDB.
func (r *UserRepository) Delete(
	ctx context.Context,
	userID string,
) error {
	ctx, cancel := context.WithTimeout(ctx, databaseTimeout)
	defer cancel()

	objectID, err := bson.ObjectIDFromHex(userID)
	if err != nil {
		return err
	}

	result, err := r.collection.DeleteOne(
		ctx,
		bson.M{
			"_id": objectID,
		},
	)

	if err != nil {
		return err
	}

	if result.DeletedCount == 0 {
		return services.ErrUserNotFound
	}

	return nil
}

func (r *UserRepository) FindByPhone(
	ctx context.Context,
	phone string,
) (*models.User, error) {
	ctx, cancel := context.WithTimeout(ctx, databaseTimeout)
	defer cancel()

	var user models.User

	err := r.collection.FindOne(
		ctx,
		bson.M{"phone": phone},
	).Decode(&user)

	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, services.ErrUserNotFound
	}

	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *UserRepository) FindByAnyEmail(
	ctx context.Context,
	email string,
) (*models.User, error) {
	ctx, cancel := context.WithTimeout(ctx, databaseTimeout)
	defer cancel()

	var user models.User

	filter := bson.M{
		"$or": bson.A{
			bson.M{"email": email},
			bson.M{"alt_email": email},
		},
	}

	err := r.collection.FindOne(ctx, filter).Decode(&user)

	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, services.ErrUserNotFound
	}

	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *UserRepository) FindByLoginIdentifier(
	ctx context.Context,
	identifier string,
) (*models.User, error) {
	ctx, cancel := context.WithTimeout(ctx, databaseTimeout)
	defer cancel()

	var user models.User

	filter := bson.M{
		"$or": bson.A{
			bson.M{"username": identifier},
			bson.M{"email": identifier},
			bson.M{"alt_email": identifier},
		},
	}

	err := r.collection.FindOne(ctx, filter).Decode(&user)

	if errors.Is(err, mongodriver.ErrNoDocuments) {
		return nil, services.ErrUserNotFound
	}

	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *UserRepository) ListCustomers(
	ctx context.Context,
	listFilter repository.CustomerListFilter,
) ([]models.User, int64, error) {
	ctx, cancel := context.WithTimeout(ctx, databaseTimeout)
	defer cancel()

	filter := bson.M{
		"role": models.RoleCustomer,
	}

	if listFilter.Status != nil {
		filter["status"] = *listFilter.Status
	}

	if listFilter.Search != "" {
		searchRegex := bson.Regex{
			Pattern: regexp.QuoteMeta(listFilter.Search),
			Options: "i",
		}

		filter["$or"] = bson.A{
			bson.M{"first_name": searchRegex},
			bson.M{"last_name": searchRegex},
			bson.M{"username": searchRegex},
			bson.M{"email": searchRegex},
			bson.M{"alt_email": searchRegex},
			bson.M{"phone": searchRegex},
		}
	}

	total, err := r.collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	findOptions := options.Find().
		SetSkip(listFilter.Skip).
		SetLimit(listFilter.Limit).
		SetSort(bson.D{
			{Key: "created_at", Value: -1},
		})

	cursor, err := r.collection.Find(
		ctx,
		filter,
		findOptions,
	)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var customers []models.User

	if err := cursor.All(ctx, &customers); err != nil {
		return nil, 0, err
	}

	return customers, total, nil
}
