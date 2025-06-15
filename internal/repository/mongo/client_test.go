package mongo

import (
	"TrackMe/internal/domain/client"
	"TrackMe/pkg/store"
	"context"
	"fmt"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"log"
	"testing"
	"time"
)

type TestDatabase struct {
	DbInstance *mongo.Database
	DbAddress  string
	container  testcontainers.Container
}

func SetupTestDatabaseWithName(name string) *TestDatabase {
	ctx, _ := context.WithTimeout(context.Background(), time.Second*60)
	container, dbInstance, dbAddr, err := createMongoContainer(ctx, name)
	if err != nil {
		log.Fatal("failed to setup test", err)
	}

	return &TestDatabase{
		container:  container,
		DbInstance: dbInstance,
		DbAddress:  dbAddr,
	}
}

func (tdb *TestDatabase) TearDown() {
	_ = tdb.container.Terminate(context.Background())
}

func NewMongoDatabase(uri string, database string) (*mongo.Database, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		return nil, err
	}

	db := client.Database(database)

	return db, nil
}

func createMongoContainer(ctx context.Context, name string) (testcontainers.Container, *mongo.Database, string, error) {
	var env = map[string]string{
		"MONGO_INITDB_ROOT_USERNAME": "root",
		"MONGO_INITDB_ROOT_PASSWORD": "pass",
		"MONGO_INITDB_DATABASE":      name,
	}
	var port = "27017/tcp"

	req := testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "mongo",
			ExposedPorts: []string{port},
			Env:          env,
		},
		Started: true,
	}
	container, err := testcontainers.GenericContainer(ctx, req)
	if err != nil {
		return container, nil, "", fmt.Errorf("failed to start container: %v", err)
	}

	p, err := container.MappedPort(ctx, "27017")
	if err != nil {
		return container, nil, "", fmt.Errorf("failed to get container external port: %v", err)
	}

	log.Println("mongo container ready and running at port: ", p.Port())

	uri := fmt.Sprintf("mongodb://root:pass@localhost:%s", p.Port())
	db, err := NewMongoDatabase(uri, name)
	if err != nil {
		return container, db, uri, fmt.Errorf("failed to establish database connection: %v", err)
	}

	return container, db, uri, nil
}

type RepositorySuite struct {
	suite.Suite
	repository   client.Repository
	testDatabase *TestDatabase
}

type SetupAllSuite interface {
	SetupSuite()
}

type TearDownAllSuite interface {
	TearDownSuite()
}

func (suite *RepositorySuite) SetupSuite() {
	suite.testDatabase = SetupTestDatabaseWithName("cliets_test_db")
	suite.repository = NewClientRepository(suite.testDatabase.DbInstance)
}

func (suite *RepositorySuite) TearDownSuite() {
	suite.testDatabase.container.Terminate(context.Background())
}

func (suite *RepositorySuite) TestGet() {
	suite.Run("when there is no record", func() {
		id := primitive.NewObjectID()

		foundClient, err := suite.repository.Get(context.Background(), id.Hex())

		suite.Equal(store.ErrorNotFound, err)
		suite.Equal(client.Entity{}, foundClient)
	})

	suite.Run("when there is record for given id", func() {
		name := "Test Client"
		email := "test@example.com"
		stage := "registration"
		isActive := true

		newClient := client.Entity{
			Name:         &name,
			Email:        &email,
			CurrentStage: &stage,
			IsActive:     &isActive,
		}

		createdClient, err := suite.repository.Update(context.Background(),
			primitive.NewObjectID().Hex(), newClient)
		suite.NoError(err)

		foundClient, err := suite.repository.Get(context.Background(), createdClient.ID)

		suite.NoError(err)
		suite.Equal(createdClient, foundClient)
		suite.Equal(*foundClient.Name, name)
		suite.Equal(*foundClient.Email, email)
	})

	suite.Run("with invalid id", func() {
		_, err := suite.repository.Get(context.Background(), "invalid-id")
		suite.Error(err)
	})
}

func (suite *RepositorySuite) TestUpdate() {
	suite.Run("create new client", func() {
		name := "New Client"
		email := "new@example.com"
		stage := "onboarding"
		source := "website"
		channel := "direct"
		isActive := true

		newClient := client.Entity{
			Name:         &name,
			Email:        &email,
			CurrentStage: &stage,
			Source:       &source,
			Channel:      &channel,
			IsActive:     &isActive,
		}

		id := primitive.NewObjectID().Hex()
		createdClient, err := suite.repository.Update(context.Background(), id, newClient)

		suite.NoError(err)
		suite.Equal(id, createdClient.ID)
		suite.Equal(*createdClient.Name, name)
		suite.Equal(*createdClient.Email, email)
		suite.Equal(*createdClient.CurrentStage, stage)
		suite.Equal(*createdClient.Source, source)
		suite.Equal(*createdClient.Channel, channel)
		suite.True(*createdClient.IsActive)
	})

	suite.Run("update existing client", func() {
		// First create a client
		name := "Original Name"
		email := "original@example.com"
		stage := "registration"

		originalClient := client.Entity{
			Name:         &name,
			Email:        &email,
			CurrentStage: &stage,
		}

		id := primitive.NewObjectID().Hex()
		_, err := suite.repository.Update(context.Background(), id, originalClient)
		suite.NoError(err)

		// Now update the client
		updatedName := "Updated Name"
		updatedStage := "completed"
		updatedSource := "referral"

		updatedClient := client.Entity{
			Name:         &updatedName,
			Email:        &email, // keep same email
			CurrentStage: &updatedStage,
			Source:       &updatedSource,
		}

		result, err := suite.repository.Update(context.Background(), id, updatedClient)

		suite.NoError(err)
		suite.Equal(id, result.ID)
		suite.Equal(*result.Name, updatedName)
		suite.Equal(*result.Email, email)
		suite.Equal(*result.CurrentStage, updatedStage)
		suite.Equal(*result.Source, updatedSource)
	})

	suite.Run("with invalid id", func() {
		name := "Test"
		clientData := client.Entity{
			Name: &name,
		}

		_, err := suite.repository.Update(context.Background(), "invalid-id", clientData)
		suite.Error(err)
	})
}

func (suite *RepositorySuite) TestList() {
	// First, clear any existing data
	_, err := suite.testDatabase.DbInstance.Collection("clients").DeleteMany(
		context.Background(), bson.M{})
	suite.NoError(err)

	// Create test clients
	suite.createTestClients()

	suite.Run("list all clients", func() {
		clients, total, err := suite.repository.List(
			context.Background(), client.Filters{}, 10, 0)

		suite.NoError(err)
		suite.Equal(3, total)
		suite.Equal(3, len(clients))
	})

	suite.Run("list with stage filter", func() {
		clients, total, err := suite.repository.List(
			context.Background(),
			client.Filters{
				Stage: "registration",
			}, 10, 0)

		suite.NoError(err)
		suite.Equal(2, total)
		suite.Equal(2, len(clients))
	})

	suite.Run("list with source filter", func() {
		clients, total, err := suite.repository.List(
			context.Background(),
			client.Filters{
				Source: "website",
			}, 10, 0)

		suite.NoError(err)
		suite.Equal(2, total)
		suite.Equal(2, len(clients))
	})

	suite.Run("list with isActive filter", func() {
		isActive := true
		clients, total, err := suite.repository.List(
			context.Background(),
			client.Filters{
				IsActive: &isActive,
			}, 10, 0)

		suite.NoError(err)
		suite.Equal(2, total)
		suite.Equal(2, len(clients))
	})

	suite.Run("list with pagination", func() {
		clients, total, err := suite.repository.List(
			context.Background(), client.Filters{}, 2, 0)

		suite.NoError(err)
		suite.Equal(3, total)        // Total is still 3
		suite.Equal(2, len(clients)) // But only 2 returned

		// Get next page
		nextClients, nextTotal, err := suite.repository.List(
			context.Background(), client.Filters{}, 2, 2)

		suite.NoError(err)
		suite.Equal(3, nextTotal)
		suite.Equal(1, len(nextClients))
	})
}

func (suite *RepositorySuite) TestCount() {
	// First, clear any existing data
	_, err := suite.testDatabase.DbInstance.Collection("clients").DeleteMany(
		context.Background(), bson.M{})
	suite.NoError(err)

	// Create test clients
	suite.createTestClients()

	suite.Run("count all clients", func() {
		count, err := suite.repository.Count(context.Background(), bson.M{})

		suite.NoError(err)
		suite.Equal(int64(3), count)
	})

	suite.Run("count with filter", func() {
		count, err := suite.repository.Count(
			context.Background(),
			bson.M{"current_stage": "registration"})

		suite.NoError(err)
		suite.Equal(int64(2), count)
	})
}

// Helper method to create test data
func (suite *RepositorySuite) createTestClients() {
	// Client 1
	name1 := "User One"
	email1 := "user1@example.com"
	stage1 := "registration"
	source1 := "website"
	isActive1 := true

	client1 := client.Entity{
		Name:         &name1,
		Email:        &email1,
		CurrentStage: &stage1,
		Source:       &source1,
		IsActive:     &isActive1,
	}

	// Client 2
	name2 := "User Two"
	email2 := "user2@example.com"
	stage2 := "registration"
	source2 := "referral"
	isActive2 := true

	client2 := client.Entity{
		Name:         &name2,
		Email:        &email2,
		CurrentStage: &stage2,
		Source:       &source2,
		IsActive:     &isActive2,
	}

	// Client 3
	name3 := "User Three"
	email3 := "user3@example.com"
	stage3 := "completed"
	source3 := "website"
	isActive3 := false

	client3 := client.Entity{
		Name:         &name3,
		Email:        &email3,
		CurrentStage: &stage3,
		Source:       &source3,
		IsActive:     &isActive3,
	}

	id1 := primitive.NewObjectID().Hex()
	id2 := primitive.NewObjectID().Hex()
	id3 := primitive.NewObjectID().Hex()

	_, err1 := suite.repository.Update(context.Background(), id1, client1)
	_, err2 := suite.repository.Update(context.Background(), id2, client2)
	_, err3 := suite.repository.Update(context.Background(), id3, client3)

	suite.NoError(err1)
	suite.NoError(err2)
	suite.NoError(err3)
}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestClientRepositorySuite(t *testing.T) {
	suite.Run(t, new(RepositorySuite))
}
