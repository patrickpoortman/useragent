// Copyright (C) 2014-2023 Miquel Sabaté Solà <mikisabate@gmail.com>
// This file is licensed under the MIT license.
// See the LICENSE file.

package useragent

import (
	"regexp"
	"strings"
	"sync"
)

var botFromSiteRegexp = regexp.MustCompile(`http[s]?://.+\.\w+`)

var (
	badBotsList []string
	badBotsOnce sync.Once
)

// LoadBadBotsYAML loads the bad bots list from YAML only once (thread-safe).
func LoadBadBotsYAML() []string {
	badBotsOnce.Do(func() {
		list, err := LoadBadBots("bad_bots.yaml")
		if err == nil {
			badBotsList = list
		}
	})
	return badBotsList
}

// Get the name of the bot from the website that may be in the given comment. If
// there is no website in the comment, then an empty string is returned.
func getFromSite(comment []string) string {
	if len(comment) == 0 {
		return ""
	}

	// Where we should check the website.
	idx := 2
	if len(comment) < 3 {
		idx = 0
	} else if len(comment) == 4 {
		idx = 3
	}

	// Pick the site.
	results := botFromSiteRegexp.FindStringSubmatch(comment[idx])
	if len(results) == 1 {
		// If it's a simple comment, just return the name of the site.
		if idx == 0 {
			return results[0]
		}

		// This is a large comment, usually the name will be in the previous
		// field of the comment.
		return strings.TrimSpace(comment[idx-1])
	}
	return ""
}

// Returns true if the info that we currently have corresponds to the Google
// or Bing mobile bot. This function also modifies some attributes in the receiver
// accordingly.
func (p *UserAgent) googleOrBingBot() bool {
	// This is a hackish way to detect
	// Google's mobile bot (Googlebot, AdsBot-Google-Mobile, etc.)
	// (See https://support.google.com/webmasters/answer/1061943)
	// and Bing's mobile bot
	// (See https://www.bing.com/webmaster/help/which-crawlers-does-bing-use-8c184ec0)
	if strings.Contains(p.ua, "Google") || strings.Contains(p.ua, "bingbot") {
		p.platform = ""
		p.undecided = true
	}
	return p.undecided
}

// Returns true if we think that it is iMessage-Preview. This function also
// modifies some attributes in the receiver accordingly.
func (p *UserAgent) iMessagePreview() bool {
	// iMessage-Preview doesn't advertise itself. We have a to rely on a hack
	// to detect it: it impersonates both facebook and twitter bots.
	// See https://medium.com/@siggi/apples-imessage-impersonates-twitter-facebook-bots-when-scraping-cef85b2cbb7d
	if !strings.Contains(p.ua, "facebookexternalhit") {
		return false
	}
	if !strings.Contains(p.ua, "Twitterbot") {
		return false
	}
	p.bot = true
	p.browser.Name = "iMessage-Preview"
	p.browser.Engine = ""
	p.browser.EngineVersion = ""
	// We don't set the mobile flag because iMessage can be on iOS (mobile) or macOS (not mobile).
	return true
}

// Set the attributes of the receiver as given by the parameters. All the other
// parameters are set to empty.
func (p *UserAgent) setSimple(name, version string, bot bool) {
	p.bot = bot
	if !bot {
		p.mozilla = ""
	}
	p.browser.Name = name
	p.browser.Version = version
	p.browser.Engine = ""
	p.browser.EngineVersion = ""
	p.os = ""
	p.localization = ""
}

// Fix some values for some weird browsers.
func (p *UserAgent) fixOther(sections []section) {
	if len(sections) > 0 {
		p.browser.Name = sections[0].name
		p.browser.Version = sections[0].version
		p.mozilla = ""
	}
}

// Checks if the given string contains any known bad bot substring (case-insensitive).
func isKnownBadBot(s string) bool {
	for _, bot := range LoadBadBotsYAML() {
		if strings.Contains(strings.ToLower(s), strings.ToLower(bot)) {
			return true
		}
	}
	return false
}

// Check if we're dealing with a bot or with some weird browser. If that is the
// case, the receiver will be modified accordingly.
func (p *UserAgent) checkBot(sections []section) {
	// If there's only one element, and it doesn't have the Mozilla string,
	// check whether this is a bot or not.
	if len(sections) == 1 && sections[0].name != "Mozilla" {
		p.mozilla = ""

		// Check whether the name matches any known bad bot substring.
		if isKnownBadBot(sections[0].name) {
			p.setSimple(sections[0].name, "", true)
			return
		}

		// Tough luck, let's try to see if it has a website in his comment.
		if name := getFromSite(sections[0].comment); name != "" {
			p.setSimple(sections[0].name, sections[0].version, true)
			return
		}

		p.setSimple(sections[0].name, sections[0].version, false)
	} else {
		for _, v := range sections {
			if name := getFromSite(v.comment); name != "" {
				results := strings.SplitN(name, "/", 2)
				version := ""
				if len(results) == 2 {
					version = results[1]
				}
				p.setSimple(results[0], version, true)
				return
			}
			// Also check each section name for known bad bots
			if isKnownBadBot(v.name) {
				p.setSimple(v.name, v.version, true)
				return
			}
		}
		p.fixOther(sections)
	}
}
