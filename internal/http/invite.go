package http

import (
	"context"
	"encoding/json"
	"errors"
	jwt "github.com/appleboy/gin-jwt/v2"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"net/http"
	"slicerapi/internal/config"
	"slicerapi/internal/db"
	"slicerapi/internal/http/ws"
	"time"
)

type reqInviteAdd struct {
	ID string `json:"id"`
}

func handleInviteJoin(c *gin.Context) {
	userID := jwt.ExtractClaims(c)["id"].(string)
	chID := c.Param("channel")

	var channel db.Channel

	ctx, _ := context.WithTimeout(context.Background(), time.Second*2)
	if err := db.Mongo.Database(config.C.MongoDB.Name).Collection("channels").FindOne(
		ctx,
		bson.M{
			"_id": chID,
		},
	).Decode(&channel); err != nil {
		if err == mongo.ErrNoDocuments {
			chk(http.StatusUnauthorized, err, c)
			return
		}
		chk(http.StatusInternalServerError, err, c)
		return
	}

	stat := http.StatusOK

	_, ok := channel.Pending[userID]
	if ok {
		delete(channel.Pending, userID)

		channel.Users[userID] = true
		ctx, _ := context.WithTimeout(context.Background(), time.Second*2)
		if _, err := db.Mongo.Database(config.C.MongoDB.Name).Collection("channels").UpdateOne(
			ctx,
			bson.M{
				"_id": channel.ID,
			},
			bson.D{{
				"$set",
				bson.D{{
					"pending",
					channel.Pending,
				}, {
					"users",
					channel.Users,
				}},
			}},
		); err != nil {
			chk(500, err, c)
			return
		}

		go func() {
			_, _ = db.Mongo.Database(config.C.MongoDB.Name).Collection("users").UpdateOne(
				ctx,
				bson.M{
					"_id": userID,
				},
				bson.D{{
					"$push",
					bson.D{{
						"channels",
						channel.ID,
					}},
				}},
			)
		}()

		c.JSON(stat, statusMessage{
			Code:    stat,
			Message: "Invite accepted.",
		})
		return
	}

	stat = http.StatusForbidden
	c.JSON(stat, statusMessage{
		Code:    stat,
		Message: "User isn't in the pending list.",
	})
}

func handleInviteAdd(c *gin.Context) {
	body := reqInviteAdd{}
	err := c.ShouldBindJSON(&body)
	chk(http.StatusBadRequest, err, c)
	if err != nil {
		return
	}

	chID := c.Param("channel")
	var channel db.Channel

	ctx, _ := context.WithTimeout(context.Background(), time.Second*2)
	if err := db.Mongo.Database(config.C.MongoDB.Name).Collection("channels").FindOne(
		ctx,
		bson.M{
			"_id": chID,
		},
	).Decode(&channel); err != nil {
		if err == mongo.ErrNoDocuments {
			chk(http.StatusUnauthorized, err, c)
			return
		}
		chk(http.StatusInternalServerError, err, c)
		return
	}

	if _, ok := channel.Users[body.ID]; ok {
		chk(http.StatusConflict, errors.New("user already joined"), c)
		return
	}

	channel.Pending[body.ID] = true

	ctx, _ = context.WithTimeout(context.Background(), time.Second*2)
	if _, err := db.Mongo.Database(config.C.MongoDB.Name).Collection("channels").UpdateOne(
		ctx,
		bson.M{
			"_id": channel.ID,
		},
		bson.D{{
			"$set",
			bson.D{{
				"pending",
				channel.Pending,
			}},
		}},
	); err != nil {
		chk(500, err, c)
		return
	}

	go func() {
		marshalled, _ := json.Marshal(ws.Message{
			Method: ws.EvtAddInvite,
			Data:   channel,
		})

		client, ok := ws.C.Clients[body.ID]
		if !ok {
			return
		}

		go func() {
			for _, conn := range client {
				conn.Send <- marshalled
			}
		}()
	}()

	stat := http.StatusCreated
	c.JSON(stat, statusMessage{
		Code:    stat,
		Message: "Invite created for user.",
	})
}
