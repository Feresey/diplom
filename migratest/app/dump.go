package app

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/Feresey/diplom/migratest/schema"
	"github.com/Feresey/diplom/migratest/schema/driver"
	"go.uber.org/zap"
)

const PGdumpPath = "pg_dump"

type Dumper struct {
	logger *zap.Logger
	c      driver.Config
	d      *driver.PostgresDriver
}

func NewDumper(
	c driver.Config,
	logger *zap.Logger,
	d *driver.PostgresDriver,
) *Dumper {
	return &Dumper{
		logger: logger.Named("dumper"),
		c:      c,
		d:      d,
	}
}

func (d *Dumper) InsertData(
	ctx context.Context,
	sp schema.SchemaPatterns,
	insertFunc func(ctx context.Context) error,
) error {
	dumped, err := d.dump(ctx, sp)
	if err != nil {
		return fmt.Errorf("create dump: %w", err)
	}

	scanner := bufio.NewScanner(strings.NewReader(dumped))
	constraintOffset := 0
	dataOffset := 0

	for scanner.Scan() {
		line := scanner.Text()
		dataOffset += len(line) + 1 // newline
		if strings.Contains(line, "Type: TABLE DATA") {
			break
		}
		if strings.Contains(line, "Type: SEQUENCE SET") {
			break
		}
	}
	constraintOffset = dataOffset
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "Type: CONSTRAINT") {
			break
		}
		constraintOffset += len(line) + 1 // newline
	}

	preData := dumped[:dataOffset]
	data := dumped[dataOffset:constraintOffset]
	postData := dumped[constraintOffset:]

	if err := d.restoreDump(ctx, preData, data, postData, insertFunc); err != nil {
		return fmt.Errorf("insert data: %w", err)
	}

	return nil
}

func (d *Dumper) dump(ctx context.Context, sp schema.SchemaPatterns) (raw string, err error) {
	var commandArgs []string
	for _, allow := range sp.Whitelist {
		commandArgs = append(commandArgs, "-n", allow)
	}
	for _, block := range sp.Blacklist {
		commandArgs = append(commandArgs, "-N", block)
	}

	var buf bytes.Buffer

	cmd := exec.CommandContext(
		ctx,
		PGdumpPath,
		append(commandArgs,
			"-d", d.c.DBName,
			"-h", d.c.Host,
			"-p", strconv.Itoa(d.c.Port),
			"-U", d.c.Credentials.Username,
			"--insert",
			// "--section=pre-data",
			// "--section=post-data",
			// добавляет команды типа DROP TABLE some_table, чтобы можно было пересоздать таблицы без индексов.
			"--clean",
			// вывод на stdout
			"-f", "-",
		)...,
	)

	// TODO this is insecure
	cmd.Env = append(os.Environ(), "PGPASSWORD="+d.c.Credentials.Password)
	cmd.Stdout = &buf

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("run pg_dump: %w", err)
	}
	return buf.String(), nil
}

func (d *Dumper) restoreDump(
	ctx context.Context,
	preData, data, postData string,
	insertData func(context.Context) error,
) error {
	if _, err := d.d.Conn.Exec(ctx, preData); err != nil {
		return fmt.Errorf("create tables: %w", err)
	}

	if _, err := d.d.Conn.Exec(ctx, data); err != nil {
		return fmt.Errorf("restore data: %w", err)
	}

	if insertData != nil {
		if err := insertData(ctx); err != nil {
			return fmt.Errorf("insert data: %w", err)
		}
	}

	if _, err := d.d.Conn.Exec(ctx, postData); err != nil {
		return fmt.Errorf("create indexes: %w", err)
	}

	return nil
}
