package http

import (
	"encoding/json"
	"fmt"
	"net/url"
)

type Link struct {
	Relation string
	Href     *url.URL
}

type Links struct {
	Links []Link
}

type linkObject struct {
	Href string `json:"href"`
}

func (links Links) MarshalJSON() ([]byte, error) {
	out := map[string]linkObject{}
	for _, link := range links.Links {
		if link.Href != nil {
			out[link.Relation] = linkObject{Href: link.Href.String()}
		}
	}
	return json.Marshal(out)
}

func (links *Links) UnmarshalJSON(input []byte) error {
	in := map[string]linkObject{}
	err := json.Unmarshal(input, &in)
	if err != nil {
		return err
	}

	for rel, obj := range in {
		href, err := url.Parse(obj.Href)
		if err != nil {
			return fmt.Errorf("'%s' is not a valid url: %w", obj.Href, err)
		}
		links.Links = append(links.Links, Link{Relation: rel, Href: href})
	}

	return nil
}

func selfLinks(self *url.URL) Links {
	return Links{Links: []Link{
		Link{Relation: "self", Href: self},
	}}
}
