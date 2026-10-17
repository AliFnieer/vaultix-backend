package vault

import (
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func optionsFindOneAndUpdateReturnAfter() *options.FindOneAndUpdateOptionsBuilder {
	return options.FindOneAndUpdate().SetReturnDocument(options.After)
}
