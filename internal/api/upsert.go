// internal/api/upsert.go

package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
)

func Upsert(c *Client, post Post, id string) (string, error) {
	ctx := context.Background()
	var res struct {
		Posts []struct {
			ID string `json:"id"`
		}
	}

	if id == "" { // create
		if err := c.Post(ctx, "posts/?source=html", postReq{Posts: []Post{post}}, &res); err != nil {
			return "", err
		}
	} else { // update
		// 1. fetch timestamp
		current, err := c.GetPost(ctx, id)
		if err != nil {
			return "", err
		}
		post.ID = id
		post.UpdatedAt = current.UpdatedAt // required lock
		post.Tags = nil                    // leave unchanged

		if err := c.Put(ctx, "posts/"+id+"/?source=html", postReq{Posts: []Post{post}}, &res); err != nil {
			return "", err
		}
	}

	if len(res.Posts) == 0 {
		// read full body again for debug
		if data, _ := io.ReadAll(c.lastBody); len(data) > 0 {
			var e map[string]any
			if json.Unmarshal(data, &e) == nil {
				return "", fmt.Errorf("ghost API error: %v", e)
			}
			return "", fmt.Errorf("ghost API raw: %s", data)
		}
		return "", fmt.Errorf("ghost API returned empty posts array")
	}
	return res.Posts[0].ID, nil
}
