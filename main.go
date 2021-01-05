package main

import (
	//"flag"
	"context"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"github.com/bwmarrin/discordgo"
	"github.com/go-redis/redis/v8"
)

var ctx = context.Background()

// Variables used for command line parameters
var (
	Token string
	rdb   *redis.Client
)
var users = []string{"carter", "brotman", "will", "claire", "gabby", "morgan", "fantasia", "mullikin", "rian"}

const guildMembersPageLimit = 1000

func main() {

	// flag.StringVar(&Token, "t", "", "Bot Token")
	// flag.Parse()

	Token := ""

	rdb = redis.NewClient(&redis.Options{
		Addr:     "redis-15701.c245.us-east-1-3.ec2.cloud.redislabs.com:15701",
		Password: "", // no password set
		DB:       0,                                  // use default DB
	})

	dg, err := discordgo.New("Bot " + Token)
	if err != nil {
		fmt.Println("error creating Discord session,", err)
		return
	}

    // dg.StateEnabled = true;
    // dg.State.TrackPresences = true;

    
	// Register the messageCreate func as a callback for MessageCreate events.
	dg.AddHandler(messageCreate)

	// In this example, we only care about receiving message events.
	dg.Identify.Intents = discordgo.MakeIntent(discordgo.IntentsGuildMessages)

	// Open a websocket connection to Discord and begin listening.
	err = dg.Open()
	if err != nil {
		fmt.Println("error opening connection,", err)
		return
	}

	// Wait here until CTRL-C or other term signal is received.
	fmt.Println("Bot is now running.  Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	// Cleanly close down the Discord session.
	dg.Close()
}

// This function will be called (due to AddHandler above) every time a new
// message is created on any channel that the authenticated bot has access to.
func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {

	// Ignore all messages created by the bot itself
	// This isn't required in this specific example but it's a good practice.
	if m.Author.ID == s.State.User.ID {
		return
    }
    // val, err := LookupGuild(ctx, s, m.GuildID)
    // if err != nil {
    //     fmt.Print(fmt.Errorf("%w", err))
    // }
    // fmt.Printf("guild: %+v", val)
    
	if len(m.Content) > 0 {
		if m.Content[0] == '%' {
			words := strings.Fields(m.Content)
			if len(words) == 3 && m.Author.ID == "227815692949258241"{
                if checkName(words[1]){
                    if _, err := strconv.Atoi(words[2]); err == nil {
                        err = rdb.Set(ctx, words[1], words[2], 0).Err()
                        if err != nil {
                            panic(err)
                        }
                        s.ChannelMessageSend(m.ChannelID, "Saved!")
                    } else {
                        s.ChannelMessageSend(m.ChannelID, "Use a number!")
                    }
                } else {
                    s.ChannelMessageSend(m.ChannelID, "Not a valid name!")
                }
			} else if len(words) == 2 {
                if checkName(words[1]){
                    val, err := rdb.Get(ctx, words[1]).Result()
                    if err == redis.Nil {
                        s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("%s is good to go", words[1]))
                    } else if err != nil {
                        panic(err)
                    } else {
                        s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("%s has missed %s times", words[1], val))
                    }
                } else {
                    s.ChannelMessageSend(m.ChannelID, "Not a valid name!")
                }
            } else if len(words) == 1 {
                var scoreboard strings.Builder
				scoreboard.Write([]byte("Workouts missed in a row:\n"))
                for _, username := range users {
                    workoutsMissed, err := rdb.Get(ctx, username).Result()
                    if err == redis.Nil {
                        scoreboard.Write([]byte(fmt.Sprintf("%s - 0\n", username)))
                    } else if err != nil {
                        panic(err)
                    } else {
                        scoreboard.Write([]byte(fmt.Sprintf("%s - %s\n", username, workoutsMissed)))
                    }
                 }

                 s.ChannelMessageSend(m.ChannelID, scoreboard.String())
			} else {
                s.ChannelMessageSend(m.ChannelID, "Syntax (scoreboard): %")
				s.ChannelMessageSend(m.ChannelID, "Syntax (see person): % [name]")
				s.ChannelMessageSend(m.ChannelID, "Syntax (set person): % [name] [#]")
			}
		}
	}
}


func checkName(name string) bool {
    for _, a := range users {
        if a == name {
           return true
        }
     }
     return false
}


// LookupGuild returns a *discordgo.Guild from the session's internal state
// cache. If the guild is not found in the state cache, LookupGuild will query
// the Discord API for the guild and add it to the state cache before returning
// it.
func LookupGuild(ctx context.Context, session *discordgo.Session, guildID string) (*discordgo.Guild, error) {
	guild, err := session.State.Guild(guildID)
	if err != nil {
		guild, err = updateStateGuilds(ctx, session, guildID)
		if err != nil {
			return nil, fmt.Errorf("unable to query guild: %w", err)
		}
	}

	return guild, nil
}

func updateStateGuilds(ctx context.Context, session *discordgo.Session, guildID string) (*discordgo.Guild, error) {
	guild, err := session.Guild(guildID)
	if err != nil {
		return nil, fmt.Errorf("error senging guild query request: %w", err)
	}

	roles, err := session.GuildRoles(guildID)
	if err != nil {
		return nil, fmt.Errorf("unable to query guild channels: %w", err)
	}

	channels, err := session.GuildChannels(guildID)
	if err != nil {
		return nil, fmt.Errorf("unable to query guild channels: %w", err)
	}

	members, err := recursiveGuildMembers(session, guildID, "", guildMembersPageLimit)
	if err != nil {
		return nil, fmt.Errorf("unable to query guild members: %w", err)
	}

	guild.Roles = roles
	guild.Channels = channels
	guild.Members = members
	guild.MemberCount = len(members)

	err = session.State.GuildAdd(guild)
	if err != nil {
		return nil, fmt.Errorf("unable to add guild to state cache: %w", err)
	}

	return guild, nil
}


func recursiveGuildMembers(
	session *discordgo.Session,
	guildID, after string,
	limit int,
) ([]*discordgo.Member, error) {
	guildMembers, err := session.GuildMembers(guildID, after, limit)
	if err != nil {
		return nil, fmt.Errorf("error sending recursive guild members request: %w", err)
	}

	if len(guildMembers) < guildMembersPageLimit {
		return guildMembers, nil
	}

	nextGuildMembers, err := recursiveGuildMembers(
		session,
		guildID,
		guildMembers[len(guildMembers)-1].User.ID,
		guildMembersPageLimit,
	)
	if err != nil {
		return nil, err
	}

	return append(guildMembers, nextGuildMembers...), nil
}