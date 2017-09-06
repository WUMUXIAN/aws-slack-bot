package main

// SlackAttachmentField defines a slack attachment field
type SlackAttachmentField struct {
	Title string `json:"title"`
	Value string `json:"value"`
	Short bool   `json:"short"`
}

// SlackAttachment defines a slack attachment
type SlackAttachment struct {
	Fallback   string                 `json:"fallback,omitempty"`
	Color      string                 `json:"color,omitempty"`
	PreText    string                 `json:"pretext,omitempty"`
	AuthorName string                 `json:"author_name,omitempty"`
	AuthorLink string                 `json:"author_link,omitempty"`
	AuthorIcon string                 `json:"author_icon,omitempty"`
	Title      string                 `json:"title,omitempty"`
	TitleLink  string                 `json:"title_link,omitempty"`
	Text       string                 `json:"text,omitempty"`
	Fields     []SlackAttachmentField `json:"fields,omitempty"`
	ImageURL   string                 `json:"image_url,omitempty"`
	ThumbURL   string                 `json:"thumb_url,omitempty"`
	Footer     string                 `json:"footer,omitempty"`
	FooterIcon string                 `json:"footer_icon,omitempty"`
	TS         int64                  `json:"ts,omitempty"`
}

// SlackAttachments defines slack attachments
type SlackAttachments struct {
	Attacments []SlackAttachment `json:"attachments"`
}
