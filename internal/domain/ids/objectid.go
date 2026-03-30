package ids

import "go.mongodb.org/mongo-driver/bson/primitive"

func DedupeObjectIDs(objectIDs []primitive.ObjectID) []primitive.ObjectID {
	seen := make(map[primitive.ObjectID]struct{}, len(objectIDs))
	result := make([]primitive.ObjectID, 0, len(objectIDs))
	for _, objectID := range objectIDs {
		if _, ok := seen[objectID]; ok {
			continue
		}
		seen[objectID] = struct{}{}
		result = append(result, objectID)
	}
	return result
}
