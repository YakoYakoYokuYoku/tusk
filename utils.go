package tusk

import (
	"github.com/lrstanley/girc"
	"regexp"
	"sort"
	"strings"
)

// This file contains misc. utilities

// BanUser will ban the specified user from a channel
func BanUser(c *girc.Client, channel string, user string) {
	c.Cmd.Ban(channel, user)
}

// BanUsers will ban multiple users from a channel
func BanUsers(c *girc.Client, channel string, users []string) {
	for _, user := range users { // For each user
		BanUser(c, channel, user) // Issue a BanUser
	}
}

// DeduplicateList will eliminate duplicates from a list
func DeduplicateList(list []string) []string {
	var itemsInList = make(map[string]bool) // Define itemsInList as a list of items. Makes it easy to determine that we've already added an item
	newList := []string{}

	for _, entry := range list { // For each entry in list
		if _, exists := itemsInList[entry]; !exists { // entry not in list
			itemsInList[entry] = true
			newList = append(newList, entry) // Add the entry
		}
	}

	sort.Strings(newList) // Sort our entries
	return newList
}

// IsAdmin will check our issuer, fullIssuer (includes ident), and host if they match our admin list
func IsAdmin(issuer, fullIssuer, host string) (userIsAdmin bool) {
	for _, admin := range Config.Users.Admins { // For each listed admin
		userIsAdmin = Matches(admin, issuer) // Check for a match against the username

		if !userIsAdmin { // User not an admin by nick
			userIsAdmin = Matches(admin, host) // Check for a match against the host (more secure in some cases)
		}

		if !userIsAdmin {
			userIsAdmin = Matches(admin, fullIssuer) // Try one last time but with full issuer
		}

		if userIsAdmin { // If this is a match
			break
		}
	}

	return
}

// IsInStringArr will check if this item is in the specified string array
func IsInStringArr(list []string, item string) bool {
	var isInArr bool

	for _, listItem := range list {
		if listItem == item {
			isInArr = true
			break
		}
	}

	return isInArr
}

// KickUser will kick the specified user from a channel
func KickUser(c *girc.Client, channel string, user string) {
	c.Cmd.Kick(channel, user, "Detected by this Narwhal for kick approval. Kicking.")
}

// KickUsers will kick multiple users from a channel
func KickUsers(c *girc.Client, channel string, users []string) {
	for _, user := range users { // For each user
		KickUser(c, channel, user) // Issue a KickUser
	}
}

// Matches is our string match function that checks our provided string against a requirement
// Such requirement can be basic globbing, regex, or exact match.
func Matches(requirement string, checking string) bool {
	var matches bool
	matchFromEnd := strings.HasPrefix(requirement, "*")       // Check if we're globbing from the start
	matchFromBeginning := strings.HasSuffix(requirement, "*") // Check if we're globbing at the end
	hasReg := strings.HasPrefix(requirement, "re:")           // Check if this is a regex based match

	if hasReg { // Is Regex
		regexMessage := strings.TrimPrefix(requirement, "re:")          // Remove the indicator this is a regex
		if regex, reErr := regexp.Compile(regexMessage); reErr == nil { // If we create our regex object and it is valid
			if regex.MatchString(checking) { // If we get a regex match
				matches = true
			}
		}
	} else if matchFromEnd || matchFromBeginning { // Has beginning or ending glob
		noGlobMatch := strings.Replace(requirement, "*", "", -1)

		if matchFromEnd && matchFromBeginning { // If we're globbing both sides, meaning a single contains
			if strings.Contains(checking, noGlobMatch) { // If our checking string contains the noGlobMatch
				matches = true
			}
		} else if matchFromEnd && !matchFromBeginning { // If we're only globbing the beginning
			if strings.HasSuffix(checking, noGlobMatch) { // If our checking string ends with noGlobMatch
				matches = true
			}
		} else if !matchFromEnd && matchFromBeginning { // If we're only globbing the ending
			if strings.HasPrefix(checking, noGlobMatch) { // If our checking string begins with noGlobMatch
				matches = true
			}
		}
	} else { // Exact match
		if checking == requirement { // If this is an exact match
			matches = true
		}
	}

	return matches
}

// ParseMessage will parse an event and return a NarwhalMessage
func ParseMessage(c *girc.Client, e girc.Event) NarwhalMessage {
	var channel string
	var command string
	var params []string
	user := e.Source.Name

	if user == "" { // User is somehow empty
		user = e.Source.Ident // Change to using Ident
	}

	fullIssuer := e.Source.Ident + "@" + e.Source.Host

	var authenticated bool

	if clientUser := c.LookupUser(user); clientUser != nil { // If we got the user
		if e.IsFromChannel() { // If this is from a channel
			channel = e.Params[0] // Channel is first param

			if channelPerms, inChannel := clientUser.Perms.Lookup(channel); inChannel { // Get the channel permissions
				authenticated = channelPerms.IsTrusted()
			}
		} else { // From a user directly (DM)
			var userInFullIssuer bool

			for _, admin := range Config.Users.Admins { // For each listed admin
				userInFullIssuer = Matches(admin, fullIssuer) // Try one last time but with full issuer

				if userInFullIssuer {
					break
				}
			}

			authenticated = userInFullIssuer
		}
	}

	message := strings.TrimSpace(e.Last())
	msgSplit := strings.Split(message, " ") // Split on whitespace

	if strings.HasPrefix(message, ".") { // Starts with .
		command = strings.Replace(msgSplit[:1][0], ".", "", -1) // Get the first item, remove .

		if len(msgSplit) > 1 {
			params = msgSplit[1:]
		}
	}

	return NarwhalMessage{
		Admin:         (IsAdmin(user, fullIssuer, e.Source.Host) && authenticated),
		Authenticated: authenticated,
		Channel:       channel,
		Command:       command,
		Host:          e.Source.Host,
		FullIssuer:    fullIssuer,
		Issuer:        user,
		Message:       e.Last(),
		MessageNoCmd:  strings.TrimSpace(strings.TrimPrefix(message, "."+command)),
		Params:        params,
	}
}

// RemoveFromStringArr will remove items from the string array
func RemoveFromStringArr(list []string, items []string) []string {
	var itemsList = make(map[string]bool) // Map of items and their add / remove state
	newList := []string{}                 // Items to retain

	for _, item := range list { // For each item in our list
		for _, itemToRemove := range items { // Items we're wanting to remove
			if itemToRemove == item { // If this item matches the one we're wanting to remove
				itemsList[itemToRemove] = true // Should remove the item
				break
			}
		}

		if _, exists := itemsList[item]; !exists { // Item shouldn't be removed
			newList = append(newList, item) // Add item to new list
		}
	}

	return newList
}

// UnbanUser will unban the specified user from a channel
func UnbanUser(c *girc.Client, channel string, user string) {
	c.Cmd.Unban(channel, user)
}

// UnbanUsers will unban multiple users from a channel
func UnbanUsers(c *girc.Client, channel string, users []string) {
	for _, user := range users {
		UnbanUser(c, channel, user) // Issue an UnbanUser
	}
}
