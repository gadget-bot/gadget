package how

// Most of this was shamelessly stolen from https://smarx.com/posts/2020/09/shove-it-up-your-bot-an-intro-to-slack-bots/

import (
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"net/http"
	"net/url"
	"regexp"

	"github.com/gadget-bot/gadget/router"

	"astuart.co/goq"

	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
)

type wikiSearch struct {
	Title string
}
type wikiQuery struct {
	Search []wikiSearch
}
type wikiSearchResult struct {
	Query wikiQuery
}

type wikiPage struct {
	Steps []string `goquery:"b.whb,text"`
}

func getInstructions(query string) string {
	req, _ := http.NewRequest("GET", "http://www.wikihow.com/api.php", nil)

	q := req.URL.Query()
	q.Add("action", "query")
	q.Add("list", "search")
	q.Add("srsearch", query)
	q.Add("format", "json")

	req.URL.RawQuery = q.Encode()

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return "Sorry, I couldn't query the wikiHow API."
	}
	defer res.Body.Close()

	var result wikiSearchResult
	json.NewDecoder(res.Body).Decode(&result)

	pages := result.Query.Search

	if len(pages) == 0 {
		return "Sorry, but I don't know how to do that."
	}

Attempt:
	for attempts := 0; attempts < 3; attempts++ {
		title := pages[rand.Intn(len(pages))].Title

		res, err = http.Get("http://www.wikihow.com/" + url.PathEscape(title))
		if err != nil {
			return "Sorry, but I couldn't fetch the wikiHow page."
		}
		defer res.Body.Close()

		if res.StatusCode == 200 {
			var page wikiPage

			err = goq.NewDecoder(res.Body).Decode(&page)
			if err != nil {
				continue
			}

			length := float64(len(page.Steps))
			// number of legitimate steps to use (3 <= howMany <= length, max of 6)
			howMany := int(math.Min(length, float64(3+rand.Intn(int(math.Min(3, length))))))

			// format as a bulleted list:
			// * _1._ first thing
			// * _2._ second thing
			// ...
			response := "*" + title + "*\n"
			for i := 0; i < howMany; i++ {
				if len(page.Steps[i]) == 0 {
					// No step to include here, try again.
					continue Attempt
				}
				response += fmt.Sprintf("_%d._ %s\n", i+1, page.Steps[i])
			}

			// add the punchline
			response += fmt.Sprintf("_%d._ Profit!\n", howMany+1)

			return response
		}
	}

	return "Sorry, but I don't know how to do that."
}

func howDoI() *router.MentionRoute {
	var pluginRoute router.MentionRoute
	// Allow anyone to use this route
	pluginRoute.Permissions = append(pluginRoute.Permissions, "*")
	pluginRoute.Name = "how.howDoI"
	pluginRoute.Pattern = `(?i)^How Do I ([^?]+)\??`
	pluginRoute.Plugin = func(api slack.Client, router router.Router, ev slackevents.AppMentionEvent, message string) {
		msgRef := slack.NewRefToMessage(ev.Channel, ev.TimeStamp)
		api.AddReaction("eyes", msgRef)

		re := regexp.MustCompile(pluginRoute.Pattern)
		results := re.FindStringSubmatch(message)
		query := results[1]

		api.PostMessage(
			ev.Channel,
			slack.MsgOptionText(getInstructions(query), false),
		)
	}
	return &pluginRoute
}

func GetMentionRoutes() []router.MentionRoute {
	return []router.MentionRoute{*howDoI()}
}
