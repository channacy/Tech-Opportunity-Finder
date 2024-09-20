package shared

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Opportunity struct {
	ID                 primitive.ObjectID `bson:"_id,omitempty"`
	Name               string             `bson:"name,omitempty"`
	Url                string             `bson:"url,omitempty"`
	Category           string             `bson:"category,omitempty"`
	CreatedAt          time.Time          `bson:"createdAt,omitempty"`
	ExpireAfterSeconds int                `bson:"expireAfterSeconds,omitempty"`
}
