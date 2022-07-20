package weather

import "github.com/yangrq1018/jerry-bot/telegram"

func Commands() []telegram.Command {
	r := NewReminder()
	subs := &subscribe{
		reminder: r,
	}
	cur := &current{
		reminder: r,
	}
	return []telegram.Command{
		subs,
		cur,
		new(forecast),
	}

}
