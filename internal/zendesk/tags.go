package zendesk

import (
	"context"
	"fmt"
)

type TagsResp struct {
	Tags  []Tag `json:"tags"`
	Meta  Meta  `json:"meta"`
	Links Links `json:"links"`
}

type Tag struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

func (c *Client) GetTags(ctx context.Context) ([]Tag, error) {
	initialUrl := fmt.Sprintf("%s/tags?page[size]=100", c.baseUrl)
	allTags := &TagsResp{}
	currentPage := &TagsResp{}

	if err := c.ApiRequest(ctx, "GET", initialUrl, nil, &currentPage); err != nil {
		return nil, fmt.Errorf("an error occured getting tags: %w", err)
	}

	allTags.Tags = append(allTags.Tags, currentPage.Tags...)

	for currentPage.Meta.HasMore {
		nextPage := &TagsResp{}
		if err := c.ApiRequest(ctx, "GET", currentPage.Links.Next, nil, &nextPage); err != nil {
			return nil, fmt.Errorf("an error occured getting the tags: %w", err)
		}

		allTags.Tags = append(allTags.Tags, nextPage.Tags...)
		currentPage = nextPage
	}

	return allTags.Tags, nil
}
