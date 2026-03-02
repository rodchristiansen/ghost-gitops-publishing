// internal/api/post.go

package api

type postReq struct {
	Posts []Post `json:"posts"`
}

type Post struct {
	ID             string      `json:"id,omitempty"`
	Title          string      `json:"title"`
	Slug           string      `json:"slug,omitempty"`
	Status         string      `json:"status,omitempty"`
	HTML           string      `json:"html"`
	FeatureImage   *string     `json:"feature_image"`
	Tags           []tagRef    `json:"tags,omitempty"`
	CustomExcerpt  string      `json:"custom_excerpt,omitempty"`
	PublishedAt    string      `json:"published_at,omitempty"`
	Visibility     string      `json:"visibility,omitempty"`
	Tiers          []TierRef   `json:"tiers,omitempty"`
	Featured       bool        `json:"featured,omitempty"`
	Authors        []AuthorRef `json:"authors,omitempty"`
	CustomTemplate string      `json:"custom_template,omitempty"`
	UpdatedAt      string      `json:"updated_at,omitempty"`
}

type tagRef struct {
	Name string `json:"name"`
	Slug string `json:"slug,omitempty"`
}

type AuthorRef struct {
	ID   string `json:"id"`
	Name string `json:"name,omitempty"`
}

type TierRef struct {
	ID   string `json:"id"`
	Name string `json:"name,omitempty"`
	Slug string `json:"slug,omitempty"`
}

func WrapTags(tags []string) []tagRef {
	out := make([]tagRef, len(tags))
	for i, t := range tags {
		out[i] = tagRef{Name: t}
	}
	return out
}

func WrapAuthors(ids []string) []AuthorRef {
	out := make([]AuthorRef, len(ids))
	for i, id := range ids {
		out[i] = AuthorRef{ID: id}
	}
	return out
}

// WrapTiers is now a no-op pass-through, since you already have the full struct:
func WrapTiers(ts []TierRef) []TierRef {
	return ts
}
