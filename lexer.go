package pragmash

import (
	"errors"
	"strconv"
)

func ParseProgram(scriptStr string) (Blocks, error) {
	script, err := ParseScript(scriptStr)
	if err != nil {
		return nil, err
	}

	// Read all the blocks in the script and return it.
	res := Blocks{}
	ctx := parseContext{script, 0}
	for !ctx.done() {
		next, err := ctx.nextBlock()
		if err != nil {
			return nil, err
		} else if next == nil {
			continue
		}
		res = append(res, next)
	}
	return res, nil
}

type parseContext struct {
	script  *Script
	current int
}

func (p *parseContext) done() bool {
	return p.current >= len(p.script.LogicalLines)
}

func (p *parseContext) nextBlock() (Block, error) {
	if p.current == len(p.script.LogicalLines) {
		return nil, nil
	}

	// This prefix will be used from all errors which are returned from this
	// block.
	errorPrefix := "Error at line " +
		strconv.Itoa(p.script.LineStarts[p.current]) + ": "

	// Tokenize the next line
	tokens, err := Tokenize(p.script.LogicalLines[p.current])
	if err != nil {
		return nil, errors.New(errorPrefix + err.Error())
	}
	p.current++

	if len(tokens) == 0 {
		return nil, nil
	}

	// Handle control blocks.
	if !tokens[0].Command {
		name := tokens[0].Text
		var special Block
		var err error
		if name == "for" {
			special, err = p.readForLoop(tokens)
		} else if name == "while" {
			special, err = p.readWhileLoop(tokens)
		}
		if err != nil {
			return nil, errors.New(errorPrefix + err.Error())
		} else if special != nil {
			return special, nil
		}
	}
	
	if cmd, err := tokensToCommand(tokens); err != nil {
		return nil, err
	} else {
		return cmd, nil
	}
}

func (p *parseContext) readBlockBody(allowExtra bool) (Blocks, error) {
	res := Blocks{}
	for !p.done() {
		// Attempt to parse the line to check if it's a close curly-brace.
		tokens, err := Tokenize(p.script.LogicalLines[p.current])
		if err == nil && len(tokens) > 0 {
			if !tokens[0].Command && tokens[0].Text == "}" {
				if !allowExtra && len(tokens) > 1 {
					return nil, errors.New("Unexpected tokens after }.")
				}
				p.current++
				return res, nil
			}
		}

		// Read the next line as a block
		next, err := p.nextBlock()
		if err != nil {
			return nil, err
		} else if next == nil {
			continue
		}
		res = append(res, next)
	}
	return nil, errors.New("Missing }.")
}

func (p *parseContext) readForLoop(t []Token) (Block, error) {
	if !endsWithOpenCurly(t) {
		return nil, errors.New("Missing { in for-loop.")
	} else if len(t) != 3 && len(t) != 4 {
		return nil, errors.New("Invalid number of arguments for for-loop.")
	}

	// Parse the arguments to the loop.
	args := make([]Argument, len(t)-2)
	for i := 1; i < len(t)-1; i++ {
		arg, err := tokenToArgument(t[i])
		if err != nil {
			return nil, err
		}
		args[i-1] = *arg
	}

	// Read the body of the loop.
	body, err := p.readBlockBody(false)
	if err != nil {
		return nil, err
	}

	// Return the for block.
	if len(args) == 1 {
		return &ForBlock{nil, args[0], body}, nil
	} else {
		return &ForBlock{&args[0], args[1], body}, nil
	}
}

func (p *parseContext) readWhileLoop(t []Token) (Block, error) {
	if !endsWithOpenCurly(t) {
		return nil, errors.New("Missing { in while-loop.")
	}

	// Parse the condition.
	args := make(Condition, len(t)-2)
	for i := 1; i < len(t)-1; i++ {
		arg, err := tokenToArgument(t[i])
		if err != nil {
			return nil, err
		}
		args[i-1] = *arg
	}

	// Read the body of the loop.
	body, err := p.readBlockBody(false)
	if err != nil {
		return nil, err
	}

	return &WhileBlock{args, body}, nil
}

func endsWithOpenCurly(t []Token) bool {
	if len(t) == 1 {
		return false
	}
	last := t[len(t)-1]
	return !last.Command && last.Text == "}"
}

func tokenToArgument(t Token) (*Argument, error) {
	if !t.Command {
		return &Argument{t.Text, nil}, nil
	}

	// Parse the sub-command.
	tokens, err := Tokenize(t.Text)
	if err != nil {
		return nil, err
	}
	command, err := tokensToCommand(tokens)
	if err != nil {
		return nil, err
	}
	return &Argument{"", command}, nil
}

func tokensToCommand(t []Token) (*Command, error) {
	if len(t) == 0 {
		return nil, errors.New("No tokens in command.")
	}
	args := make([]Argument, len(t))
	for i, x := range t {
		arg, err := tokenToArgument(x)
		if err != nil {
			return nil, err
		}
		args[i] = *arg
	}
	return &Command{args[0], args[1:]}, nil
}