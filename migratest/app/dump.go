package app

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"

	"github.com/Feresey/diplom/migratest/schema"
	"github.com/Feresey/diplom/migratest/schema/driver"
)

const PGdumpPath = "pg_dump"

type DumperConfig struct{}

type Dumper struct {
	c driver.Config
	d *driver.PostgresDriver
}

func NewDumper(
	c driver.Config,
	d *driver.PostgresDriver,
	cc schema.Config,
) *Dumper {
	return &Dumper{
		c: c,
		d: d,
	}
}

func (d *Dumper) Dump(ctx context.Context) (preData, postData string, err error) {
	preDataFile, err := os.CreateTemp("", "pre-data-dump-*.sql")
	if err != nil {
		return "", "", fmt.Errorf("create pre-data temp file: %w", err)
	}
	_ = preDataFile.Close()

	postDataFile, err := os.CreateTemp("", "post-data-dump-*.sql")
	if err != nil {
		return "", "", fmt.Errorf("create post-data temp file: %w", err)
	}
	_ = postDataFile.Close()

	commandArgs := []string{
		"-d", d.c.DBName,
		"-h", d.c.Host,
		"-p", strconv.Itoa(d.c.Port),
		"-U", d.c.Credentials.Username,
		"-f", preDataFile.Name(),
	}
	env := append(os.Environ(), "PGPASSWORD="+d.c.Credentials.Password)

	cmdPreData := exec.CommandContext(
		ctx,
		PGdumpPath,
		append(commandArgs, "--section=pre-data")...,
	)
	cmdPreData.Env = env

	if err := cmdPreData.Run(); err != nil {
		return "", "", fmt.Errorf("fetch pre-data: %w", err)
	}

	cmdPostData := exec.CommandContext(
		ctx,
		PGdumpPath,
		append(commandArgs, "--section=post-data")...,
	)
	cmdPostData.Env = env

	if err := cmdPostData.Run(); err != nil {
		return "", "", fmt.Errorf("fetch post-data: %w", err)
	}

	return preDataFile.Name(), postDataFile.Name(), nil
}

func (d *Dumper) RestoreDump(ctx context.Context, preData, insertData, postData string) error {
	raw, err := os.ReadFile(preData)
	if err != nil {
		return fmt.Errorf("read pre-data script: %w", err)
	}
	if _, err := d.d.Conn.Exec(ctx, string(raw)); err != nil {
		return fmt.Errorf("create tables: %w", err)
	}

	raw, err = os.ReadFile(insertData)
	if err != nil {
		return fmt.Errorf("read data insert script: %w", err)
	}
	if _, err := d.d.Conn.Exec(ctx, string(raw)); err != nil {
		return fmt.Errorf("insert data: %w", err)
	}

	raw, err = os.ReadFile(postData)
	if err != nil {
		return fmt.Errorf("read post-data script: %w", err)
	}
	if _, err := d.d.Conn.Exec(ctx, string(raw)); err != nil {
		return fmt.Errorf("create indexes: %w", err)
	}

	return nil
}
