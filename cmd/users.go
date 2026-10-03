package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/m1k1o/neko-rooms/internal/auth"
	"github.com/m1k1o/neko-rooms/internal/store"
)

// users command helps to recover access, e.g.:
//
//	docker exec neko-rooms /app/bin/neko_rooms users reset-password admin 'new-password'
func init() {
	command := &cobra.Command{
		Use:   "users",
		Short: "manage neko-rooms user accounts",
	}

	command.PersistentFlags().String("data_dir", "./data", "directory for the database")
	if err := viper.BindPFlag("users_data_dir", command.PersistentFlags().Lookup("data_dir")); err != nil {
		log.Panic().Err(err).Msg("unable to bind flag")
	}

	open := func() *store.Store {
		dir := viper.GetString("users_data_dir")
		// fall back to the same env variable the server uses
		if env := os.Getenv("NEKO_ROOMS_DATA_DIR"); env != "" && !command.PersistentFlags().Changed("data_dir") {
			dir = env
		}
		s, err := store.Open(dir)
		if err != nil {
			fmt.Fprintln(os.Stderr, "unable to open database:", err)
			os.Exit(1)
		}
		return s
	}

	fail := func(err error) {
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
	}

	command.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "list all users",
		Run: func(cmd *cobra.Command, args []string) {
			s := open()
			defer s.Close()

			users, err := s.ListUsers()
			fail(err)

			tw := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
			fmt.Fprintln(tw, "ID\tUSERNAME\tROLE\tDISABLED\tCREATED")
			for _, u := range users {
				fmt.Fprintf(tw, "%d\t%s\t%s\t%v\t%s\n", u.ID, u.Username, u.Role, u.Disabled, u.CreatedAt.Format("2006-01-02"))
			}
			tw.Flush()
		},
	})

	command.AddCommand(&cobra.Command{
		Use:   "create <username> <password> [admin|user]",
		Short: "create a user (admin by default)",
		Args:  cobra.RangeArgs(2, 3),
		Run: func(cmd *cobra.Command, args []string) {
			s := open()
			defer s.Close()

			role := store.RoleAdmin
			if len(args) == 3 {
				role = store.Role(args[2])
			}
			if !role.Valid() {
				fail(fmt.Errorf("invalid role %q", role))
			}
			fail(auth.ValidateUsername(args[0]))

			hash, err := auth.HashPassword(args[1])
			fail(err)

			u := &store.User{Username: args[0], DisplayName: args[0], PasswordHash: hash, Role: role, RoomLimit: -1}
			if role == store.RoleAdmin {
				u.RoomLimit = 0
			}
			fail(s.CreateUser(u))
			fmt.Printf("created %s %q\n", role, u.Username)
		},
	})

	command.AddCommand(&cobra.Command{
		Use:   "reset-password <username> <password>",
		Short: "set a new password, enable the account and sign it out everywhere",
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			s := open()
			defer s.Close()

			u, err := s.GetUserByUsername(args[0])
			fail(err)

			u.PasswordHash, err = auth.HashPassword(args[1])
			fail(err)
			u.Disabled = false
			fail(s.UpdateUser(u))
			fail(s.DeleteUserSessions(u.ID, ""))
			fmt.Printf("password of %q has been reset\n", u.Username)
		},
	})

	command.AddCommand(&cobra.Command{
		Use:   "set-role <username> <admin|user>",
		Short: "change the role of a user",
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			s := open()
			defer s.Close()

			u, err := s.GetUserByUsername(args[0])
			fail(err)

			u.Role = store.Role(args[1])
			if !u.Role.Valid() {
				fail(fmt.Errorf("invalid role %q", u.Role))
			}
			fail(s.UpdateUser(u))
			fmt.Printf("%q is now %s\n", u.Username, u.Role)
		},
	})

	root.AddCommand(command)
}
