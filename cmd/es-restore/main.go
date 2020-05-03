package main

import (
	"context"
	"log"
	"time"

	"github.com/ic3network/mccs-alpha-api/global"
	"github.com/ic3network/mccs-alpha-api/internal/app/repository/es"
	"github.com/ic3network/mccs-alpha-api/internal/app/repository/mongo"
	"github.com/ic3network/mccs-alpha-api/internal/app/types"
	"go.mongodb.org/mongo-driver/bson"
)

func main() {
	global.Init()
	restoreUser()
	restoreEntity()
	restoreTag()
}

func restoreUser() {
	log.Println("start restoring users")
	startTime := time.Now()

	// Don't incluse deleted item.
	filter := bson.M{
		"deletedAt": bson.M{"$exists": false},
	}

	cur, err := mongo.DB().Collection("users").Find(context.TODO(), filter)
	if err != nil {
		log.Fatal(err)
	}

	counter := 0
	for cur.Next(context.TODO()) {
		var u types.User
		err := cur.Decode(&u)
		if err != nil {
			log.Fatal(err)
		}
		// Add the user to elastic search.
		{
			userID := u.ID.Hex()
			uRecord := types.UserESRecord{
				UserID:    userID,
				FirstName: u.FirstName,
				LastName:  u.LastName,
				Email:     u.Email,
			}
			_, err = es.Client().Index().
				Index("users").
				Id(userID).
				BodyJson(uRecord).
				Do(context.Background())
		}
		counter++
	}
	if err := cur.Err(); err != nil {
		log.Fatal(err)
	}
	cur.Close(context.TODO())

	log.Printf("count %v\n", counter)
	log.Printf("took  %v\n\n", time.Now().Sub(startTime))
}

func restoreEntity() {
	log.Println("start restoring entities")
	startTime := time.Now()

	filter := bson.M{
		"deletedAt": bson.M{"$exists": false},
	}

	cur, err := mongo.DB().Collection("entities").Find(context.TODO(), filter)
	if err != nil {
		log.Fatal(err)
	}
	counter := 0
	for cur.Next(context.TODO()) {
		var b types.Entity
		err := cur.Decode(&b)
		if err != nil {
			log.Fatal(err)
		}
		// Add the entity to elastic search.
		{
			entityID := b.ID.Hex()
			uRecord := types.EntityESRecord{
				ID:              entityID,
				Name:            b.EntityName,
				Email:           b.Email,
				Offers:          b.Offers,
				Wants:           b.Wants,
				LocationCity:    b.LocationCity,
				LocationCountry: b.LocationCountry,
				Status:          b.Status,
				Categories:      b.Categories,
			}
			_, err = es.Client().Index().
				Index("entities").
				Id(entityID).
				BodyJson(uRecord).
				Do(context.Background())
		}
		counter++
	}
	if err := cur.Err(); err != nil {
		log.Fatal(err)
	}
	cur.Close(context.TODO())

	log.Printf("count %v\n", counter)
	log.Printf("took  %v\n\n", time.Now().Sub(startTime))
}

func restoreTag() {
	log.Println("start restoring tag")
	startTime := time.Now()

	// Don't incluse deleted item.
	filter := bson.M{
		"deletedAt": bson.M{"$exists": false},
	}

	cur, err := mongo.DB().Collection("tags").Find(context.TODO(), filter)
	if err != nil {
		log.Fatal(err)
	}
	counter := 0
	for cur.Next(context.TODO()) {
		var t types.Tag
		err := cur.Decode(&t)
		if err != nil {
			log.Fatal(err)
		}
		// Add the tag to elastic search.
		{
			tagID := t.ID.Hex()
			uRecord := types.TagESRecord{
				TagID:        tagID,
				Name:         t.Name,
				OfferAddedAt: t.OfferAddedAt,
				WantAddedAt:  t.WantAddedAt,
			}
			_, err = es.Client().Index().
				Index("tags").
				Id(tagID).
				BodyJson(uRecord).
				Do(context.Background())
		}
		counter++
	}
	if err := cur.Err(); err != nil {
		log.Fatal(err)
	}
	cur.Close(context.TODO())

	log.Printf("count %v\n", counter)
	log.Printf("took  %v\n\n", time.Now().Sub(startTime))
}
