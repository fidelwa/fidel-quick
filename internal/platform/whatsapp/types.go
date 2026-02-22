package whatsapp

// --- Incoming webhook payload types ---

type WebhookPayload struct {
	Object string  `json:"object"`
	Entry  []Entry `json:"entry"`
}

type Entry struct {
	ID      string   `json:"id"`
	Changes []Change `json:"changes"`
}

type Change struct {
	Field string `json:"field"`
	Value Value  `json:"value"`
}

type Value struct {
	MessagingProduct string    `json:"messaging_product"`
	Metadata         Metadata  `json:"metadata"`
	Contacts         []Contact `json:"contacts"`
	Messages         []Message `json:"messages"`
}

type Metadata struct {
	DisplayPhoneNumber string `json:"display_phone_number"`
	PhoneNumberID      string `json:"phone_number_id"`
}

type Contact struct {
	Profile Profile `json:"profile"`
	WaID    string  `json:"wa_id"`
}

type Profile struct {
	Name string `json:"name"`
}

type Message struct {
	From        string       `json:"from"`
	ID          string       `json:"id"`
	Timestamp   string       `json:"timestamp"`
	Type        string       `json:"type"`
	Text        *Text        `json:"text,omitempty"`
	Image       *Image       `json:"image,omitempty"`
	Interactive *Interactive `json:"interactive,omitempty"`
}

type Text struct {
	Body string `json:"body"`
}

type Image struct {
	ID       string `json:"id"`
	MIMEType string `json:"mime_type"`
	SHA256   string `json:"sha256"`
	Caption  string `json:"caption"`
}

type Interactive struct {
	Type      string     `json:"type"`
	ListReply *ListReply `json:"list_reply,omitempty"`
}

type ListReply struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
}

// --- Outgoing message types ---

type SendTextRequest struct {
	MessagingProduct string      `json:"messaging_product"`
	To               string      `json:"to"`
	Type             string      `json:"type"`
	Text             *TextBody   `json:"text,omitempty"`
	Interactive      *OutInteractive `json:"interactive,omitempty"`
}

type TextBody struct {
	Body string `json:"body"`
}

type OutInteractive struct {
	Type   string       `json:"type"`
	Header *OutHeader   `json:"header,omitempty"`
	Body   OutBody      `json:"body"`
	Action OutAction    `json:"action"`
}

type OutHeader struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type OutBody struct {
	Text string `json:"text"`
}

type OutAction struct {
	Button   string       `json:"button"`
	Sections []OutSection `json:"sections"`
}

type OutSection struct {
	Title string      `json:"title"`
	Rows  []OutRow    `json:"rows"`
}

type OutRow struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
}

// MediaURLResponse is the response from the media URL endpoint.
type MediaURLResponse struct {
	URL string `json:"url"`
}
