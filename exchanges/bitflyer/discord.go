package bitflyer

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

const (
	DISCORDPREFIX = "."
)

func (p *Client) Interactive() {
	if p.Discord == nil {
		return
	}

	p.Discord.AddHandler(p.interctive)
	if err := p.Discord.Open(); err != nil {
		p.SetError(false, err)
		return
	}
	defer p.Discord.Close()
}

// interctive send discord into Interactive
func (p *Client) interctive(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Ignore all messages created by the bot itself
	// This isn't required in this specific example but it's a good practice.
	if m.Author.ID == s.State.User.ID {
		return
	}

	switch {
	case m.Content == ".":
		if _, err := s.ChannelMessageSend(
			m.ChannelID,
			fmt.Sprintf("%s", options()),
		); err != nil {
			p.SetError(false, err)
			return
		}

	case strings.HasPrefix(m.Content, DISCORDPREFIX+"bf"):
		if _, err := s.ChannelMessageSend(
			m.ChannelID,
			fmt.Sprintf(
				"[%s]\nhas %.4f\nRemainOrder: %d, ResetOrder: %s\nRemain: %d, Reset: %s\nDelay: %v",
				time.Now().Format("15:04:05"),
				p.O.Size(),
				p.Private.RemainForOrder,
				p.Private.ResetForOrder.Format("15:04:05"),
				p.Private.Remain,
				p.Private.Reset.Format("15:04:05"),
				p.E.Delay)); err != nil {
			p.SetError(false, err)
			return
		}

	case strings.HasPrefix(m.Content, DISCORDPREFIX+"o"):
		var minutes = 1
		num := strings.TrimLeft(m.Content, DISCORDPREFIX+"o")
		if num != "" {
			minutes, _ = strconv.Atoi(num)
		}
		orders, err := p.GetOrdersInfo(DBTABLEORDERSINFO, time.Now().Add(-time.Duration(minutes)*time.Minute), time.Now())
		if err != nil {
			p.SetError(false, err)
			return
		}
		if len(orders) < 1 {
			return
		}
		str := make([]string, len(orders)+1)
		for i := range orders {
			str[i] = orders[i].String()
		}
		str[len(orders)] = orders[0].Columns()

		if _, err := s.ChannelMessageSend(
			m.ChannelID,
			fmt.Sprintf(
				"[%s]\n%s\n-> range before %d minutes to present, aggrigate 1 hour.",
				time.Now().Format("15:04:05"),
				strings.Join(str, "\n"),
				minutes)); err != nil {
			p.SetError(false, err)
			return
		}

	case strings.HasPrefix(m.Content, DISCORDPREFIX+"sizeup"):
		p.Setting.SizeUp()
		if _, err := s.ChannelMessageSend(
			m.ChannelID,
			fmt.Sprintf(
				"[%s]\nsuccess!\norder sizes: %v",
				time.Now().Format("15:04:05"),
				p.Setting.Size)); err != nil {
			fmt.Println(err)
			return
		}

	case strings.HasPrefix(m.Content, DISCORDPREFIX+"sizedown"):
		p.Setting.SizeDown()
		if _, err := s.ChannelMessageSend(
			m.ChannelID,
			fmt.Sprintf(
				"[%s]\nsuccess!\norder sizes: %v",
				time.Now().Format("15:04:05"),
				p.Setting.Size)); err != nil {
			p.SetError(false, err)
			return
		}

	case strings.HasPrefix(m.Content, DISCORDPREFIX+"diffup"):
		p.Setting.DiffUp()
		if _, err := s.ChannelMessageSend(
			m.ChannelID,
			fmt.Sprintf(
				"[%s]\nsuccess!\nignition change price: %.f",
				time.Now().Format("15:04:05"),
				p.E.Price*p.Setting.DiffRatio)); err != nil {
			fmt.Println(err)
			return
		}

	case strings.HasPrefix(m.Content, DISCORDPREFIX+"diffdown"):
		p.Setting.DiffDown()
		if _, err := s.ChannelMessageSend(
			m.ChannelID,
			fmt.Sprintf(
				"[%s]\nsuccess!\nignition change price: %.f",
				time.Now().Format("15:04:05"),
				p.E.Price*p.Setting.DiffRatio)); err != nil {
			p.SetError(false, err)
			return
		}

	case strings.HasPrefix(m.Content, DISCORDPREFIX+"fix"):
		o, err := p.ClosePositions()
		if err != nil {
			p.SetError(true, err)
			return
		}
		p.O.Set(*o)
		if _, err := s.ChannelMessageSend(
			m.ChannelID,
			fmt.Sprintf(
				"[%s]\nfix ordered!\n%s %.f x %.4f",
				time.Now().Format("15:04:05"),
				o.Side, p.E.Price, o.Size)); err != nil {
			p.SetError(false, err)
			return
		}

	case strings.HasPrefix(m.Content, DISCORDPREFIX+"killall"):
		if _, err := s.ChannelMessageSend(
			m.ChannelID,
			fmt.Sprintf(
				"[%s]\nkill this program after 3sec!",
				time.Now().Format("15:04:05"))); err != nil {
			p.SetError(false, err)
			return
		}
		time.Sleep(3 * time.Second)
		p.Logger.Fatal("killall order from Discord")

	}
}

func options() string {
	return fmt.Sprintf(
		`help options: <> is veriable.
	. is help, return option commands

	.bf is size & API remain
	.o is summary of orders, .o<n> return the range before <n> minutes to the present 

	.sizeup is order min size + 0.01
	.sizedown is order min size - 0.01

	.diffup is ignition price ratio + 0.00001
	.diffdown is ignition price ratio - 0.00001

	.fix is fix order
	.killall is kill process
`,
	)
}
