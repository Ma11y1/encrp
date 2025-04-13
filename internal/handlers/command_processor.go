package handlers

import (
	"bufio"
	"context"
	"encrp/internal/config"
	"encrp/internal/errors"
	"encrp/internal/logger"
	"encrp/internal/services"
	"encrp/internal/storage"
	"fmt"
	"os"
	"strings"
	"time"
)

type CommandProcessorHandler struct {
	config            *config.Config
	handlers          *Container
	services          *services.Container
	consoleReader     *bufio.Reader
	rootName          string
	position          string
	positionSeparator string
}

func NewCommandProcessorHandler(cfg *config.Config, handlers *Container, services *services.Container) *CommandProcessorHandler {
	return &CommandProcessorHandler{
		config:            cfg,
		handlers:          handlers,
		services:          services,
		consoleReader:     bufio.NewReader(os.Stdin),
		positionSeparator: cfg.Storage.PathSeparator(),
	}
}

func (c *CommandProcessorHandler) Start(ctx context.Context) error {
	st := c.services.Storage.GetStorage()
	if st == nil ||
		st.Data() == nil ||
		st.Data().Name() == "" {
		return errors.New("CommandProcessorHandler.Start()", "No loaded storage or it is invalid")
	}
	c.rootName = st.Data().Name()

	for {
		if err := ctx.Err(); err != nil {
			return errors.Wrap(err, "CommandProcessorHandler.Start()", "Termination of execution by context")
		}

		command, err := c.input()
		if err != nil {
			logger.Warnf("CommandProcessorHandler.Start()", "Error reading input command: %v", err)
			fmt.Println("Error reading command: ", err)
		}
		if command == "" {
			continue
		}

		if err = ctx.Err(); err != nil {
			return errors.Wrap(err, "CommandProcessorHandler.Start()", "Termination of execution by context")
		}

		tokens := c.getTokens(command)
		switch tokens[0] {
		case "info":
			c.handleShowInfo()
		case "to":
			if len(tokens) < 2 {
				logger.Warn("CommandProcessorHandler.Start()", "Invalid syntax command: 'to <src>'")
				fmt.Println("Invalid syntax command: 'to <src>'")
				break
			}
			err = c.handleMoveToPath(tokens[1])
		case "ls", "list":
			err = c.handleShowChildrenList()
		case "n", "new":
			path := ""
			if len(tokens) > 1 {
				path = tokens[1]
			}
			err = c.handleCreateNode(path)
		case "rm", "remove":
			if len(tokens) < 2 {
				logger.Warn("CommandProcessorHandler.Start()", "Invalid syntax command: 'remove(rm) <src>")
				fmt.Println("Invalid syntax command: 'remove(rm) <src>'")
				break
			}
			err = c.handleDeleteNode(tokens[1])
		case "sh", "show":
			path := ""
			if len(tokens) > 1 {
				path = tokens[1]
			}
			err = c.handleShowNode(path)
		case "shw", "showword":
			if len(tokens) < 2 {
				logger.Warn("CommandProcessorHandler.Start()", "Invalid syntax command: 'showword(shw) <word>'")
				fmt.Println("Invalid syntax command: 'showword(shw) <word>'")
				break
			}
			err = c.handleShowNodesByWord(tokens[1])
		case "dt", "data":
			path, key, value := "", "", ""
			if len(tokens) > 1 {
				path = tokens[1]
			}
			if len(tokens) > 2 {
				key = tokens[2]
			}
			if len(tokens) > 3 {
				value = tokens[3]
			}
			err = c.handleChangeData(path, key, value)
		case "ch", "change":
			if len(tokens) <= 1 {
				logger.Warn("CommandProcessorHandler.Start()", "Invalid syntax command: 'change(ch) <src>")
				fmt.Println("Invalid syntax command: 'change(ch) <src>'")
				break
			}
			err = c.handleChangeNode(tokens[1])
		case "mv", "move":
			if len(tokens) < 2 {
				logger.Warn("CommandProcessorHandler.Start()", "Invalid syntax command: 'move(mv) <src> <dst> or move(mv) <dst>")
				fmt.Println("Invalid syntax command: 'move(mv) <src> <dst> or move(mv) <dst>'")
				break
			}
			srcPath := ""
			dstPath := tokens[1]
			if len(tokens) > 2 {
				srcPath = tokens[1]
				dstPath = tokens[2]
			}
			err = c.handleMoveNode(srcPath, dstPath)
		case "addtag":
			if len(tokens) < 2 {
				logger.Warn("CommandProcessorHandler.Start()", "Invalid syntax command: 'addtag <tag> or addtag <src> <tag>")
				fmt.Println("Invalid syntax command: 'addtag <tag> or addtag <src> <tag>'")
				break
			}
			src, tag := "", ""
			if len(tokens) > 2 {
				src = tokens[1]
				tag = tokens[2]
			} else if len(tokens) > 1 {
				tag = tokens[1]
			}
			err = c.handleAddNodeTags(src, []string{tag})
		case "addtags": // only current node
			if len(tokens) < 2 {
				logger.Warn("CommandProcessorHandler.Start()", "Invalid syntax command: 'addtags <tags...>")
				fmt.Println("Invalid syntax command: 'addtags <tags...>")
				break
			}
			err = c.handleAddNodeTags("", tokens[1:])
		case "rmtag", "removetag":
			if len(tokens) < 2 {
				logger.Warn("CommandProcessorHandler.Start()", "Invalid syntax command: 'removetag(rmtag) <tag> or removetag(rmtag) <src> <tag>")
				fmt.Println("Invalid syntax command: 'removetag(rmtag) <tag>' or removetag(rmtag) <src> <tag>")
				break
			}
			src, tag := "", ""
			if len(tokens) > 2 {
				src = tokens[1]
				tag = tokens[2]
			} else if len(tokens) > 1 {
				tag = tokens[1]
			}
			err = c.handleRemoveNodeTags(src, []string{tag})
		case "rmtags", "removetags": // only current node
			if len(tokens) < 2 {
				logger.Warn("CommandProcessorHandler.Start()", "Invalid syntax command: 'removetags(rmtags) <tags...>")
				fmt.Println("Invalid syntax command: 'removetags(rmtags) <tags...>'")
				break
			}
			err = c.handleRemoveNodeTags("", tokens[1:])
		case "shtags", "showtags":
			if len(tokens) < 2 {
				logger.Warn("CommandProcessorHandler.Start()", "Invalid syntax command: 'showtags(shtags) <tag...>")
				fmt.Println("Invalid syntax command: 'showtags(shtags) <tag...>'")
				break
			}
			err = c.handleShowNodesByTags(tokens[1:])
		case "save":
			path := ""
			if len(tokens) > 1 {
				path = tokens[1]
			}
			err = c.handleSave(path)
		case "e", "exit":
			fmt.Println("Exiting...")
			return nil
		default:
			logger.Warnf("CommandProcessorHandler.Start()", "Invalid command %s", command)
			fmt.Printf("Invalid command %s\n", command)
		}

		if err != nil {
			logger.Warnf("CommandProcessorHandler.Start()", "Error processing command %s: %s", command, err)
			fmt.Println(err)
		}
	}
}

func (c *CommandProcessorHandler) input() (string, error) {
	prompt := c.rootName
	if c.position != "" {
		prompt += c.positionSeparator + c.position
	}
	fmt.Print(prompt, "> ")
	line, err := c.consoleReader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(line), nil
}

func (c *CommandProcessorHandler) handleShowInfo() {
	s := c.services.Storage.GetStorage()
	lastDevice := s.LastDevice()
	currentDevice := c.config.Device

	fmt.Printf("[%s] v.%s\n"+
		"Timestamps:\n\tcreate: %s\n\topen: %s\n\tsave: %s\n"+
		"Current device: %s\n\tOS: %s, Arch: %s, Platform: %s (v%s, %s)\n\tHost: %s, CPU: %d, RAM: %s, Disk: %s\n"+
		"Last device: %s\n\tOS: %s, Arch: %s, Platform: %s (v%s, %s)\n\tHost: %s, CPU: %d, RAM: %s, Disk: %s\n",
		s.Type(), s.Version(),
		time.Unix(s.TsCreate(), 0).Format("2006-01-02 15:04:05 -0700"),
		time.Unix(s.TsOpen(), 0).Format("2006-01-02 15:04:05 -0700"),
		time.Unix(s.TsSave(), 0).Format("2006-01-02 15:04:05 -0700"),
		currentDevice.HostID, currentDevice.OS, currentDevice.Architecture, currentDevice.Platform,
		currentDevice.PlatformVersion, currentDevice.PlatformFamily, currentDevice.Hostname,
		currentDevice.CPUCores, currentDevice.TotalRAM, currentDevice.TotalDisk,
		lastDevice.HostID(), lastDevice.OS(), lastDevice.Architecture(), lastDevice.Platform(),
		lastDevice.PlatformVersion(), lastDevice.PlatformFamily(), lastDevice.Hostname(),
		lastDevice.CPUCores(), lastDevice.TotalRAM(), lastDevice.TotalDisk(),
	)
}

func (c *CommandProcessorHandler) handleMoveToPath(path string) error {
	if path == "" {
		return fmt.Errorf("path is empty")
	}
	if path == "/" {
		c.position = ""
		return nil
	}

	newPosition := c.createPath(path)
	if !c.services.Storage.HasNode(newPosition) {
		return fmt.Errorf("path '%s' does not exist", newPosition)
	}
	c.position = newPosition
	return nil
}

func (c *CommandProcessorHandler) handleShowChildrenList() error {
	str := c.services.Storage

	node, err := str.GetNode(c.position)
	if err != nil {
		logger.Warnf("CommandProcessorHandler.handleShowChildrenList()", "Failed to get node by path '%s': %v", c.position, err)
		return fmt.Errorf("failed to get current node by path '%s'", c.position)
	}

	children := node.Children().Values()
	if len(children) == 0 {
		fmt.Println("node is empty")
		return nil
	}

	for i, child := range children {
		fmt.Printf("%d. [%s] [children: %d] [create: %s] [modify: %s] [tags: %s] [descr: %s]\n",
			i+1,
			child.Name(),
			child.Children().Count(),
			time.Unix(child.TsCreate(), 0).Format("2006-01-02 15:04:05"),
			time.Unix(child.TsModify(), 0).Format("2006-01-02 15:04:05"),
			strings.Join(child.Tags(), ";"),
			child.Description(),
		)
	}

	return nil
}

func (c *CommandProcessorHandler) handleShowNode(path string) error {
	str := c.services.Storage

	path = c.createPath(path)
	node, err := str.GetNode(path)
	if err != nil {
		logger.Warnf("CommandProcessorHandler.handleShowNode()", "Failed to get node by path '%s': %v", path, err)
		return fmt.Errorf("failed to get node by path '%s', node not found", path)
	}
	c.showNode(node)

	return nil
}

func (c *CommandProcessorHandler) handleShowNodesByWord(word string) error {
	if word == "" {
		return fmt.Errorf("word is empty")
	}

	str := c.services.Storage
	err := str.WalkNodes(func(node *storage.Node) {
		if strings.Contains(node.Name(), word) || strings.Contains(node.Description(), word) {
			c.showNode(node)
		}
	})
	if err != nil {
		return fmt.Errorf("failed to walk nodes: %v", err)
	}

	return nil
}

func (c *CommandProcessorHandler) showNode(node *storage.Node) {
	fmt.Printf("--------------- [%s] ---------------\n\tcreate: %s\n\tmodify: %s\n",
		node.Name(),
		time.Unix(node.TsCreate(), 0).Format("2006-01-02 15:04:05"),
		time.Unix(node.TsModify(), 0).Format("2006-01-02 15:04:05"),
	)

	tags := node.Tags()
	if len(tags) > 0 {
		fmt.Println("\ttags: |", strings.Join(tags, ";"), "|")
	}

	if node.Description() != "" {
		fmt.Println("\tdescription: ", node.Description())
	}

	fmt.Println("\tchildren: ", node.Children().Count())
	nodeChildren := node.Children()
	if nodeChildren.Count() > 0 {
		for i, name := range nodeChildren.Names() {
			child := nodeChildren.Get(name)
			if child == nil {
				continue
			}
			fmt.Printf("\t\t%d. [%s] [children: %d] [create: %s] [modify: %s] [tags: %s] descr: %s\n",
				i+1,
				child.Name(),
				child.Children().Count(),
				time.Unix(child.TsCreate(), 0).Format("2006-01-02 15:04:05"),
				time.Unix(child.TsModify(), 0).Format("2006-01-02 15:04:05"),
				strings.Join(child.Tags(), ";"),
				child.Description(),
			)
		}
	}

	dataKeys := node.Data().Keys()
	if len(dataKeys) > 0 {
		fmt.Println("\tdata: ")
		nodeData := node.Data()
		for _, key := range dataKeys {
			fmt.Printf("\t\t%s: %s\n", key, nodeData.Get(key))
		}
	}
	fmt.Println("------------------------------------")
}

func (c *CommandProcessorHandler) handleCreateNode(path string) error {
	str := c.services.Storage
	targetPath := c.createPath(path)

	fmt.Print("name>> ")
	name, err := c.consoleReader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read name: %v", err)
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("name is empty")
	}
	if strings.ContainsAny(name, c.positionSeparator+"\"") {
		return fmt.Errorf("invalid character in node name")
	}

	fullPath := targetPath
	if fullPath != "" {
		fullPath += c.positionSeparator
	}
	fullPath += name

	if str.HasNode(fullPath) {
		fmt.Print("Node already exists. Overwrite? (Y/N): ")
		res, err := c.consoleReader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read response to node rewrite question: %v", err)
		}
		res = strings.TrimSpace(res)
		if res != "Y" && res != "y" {
			return nil
		}

		if err = str.WipeNode(fullPath); err != nil {
			return fmt.Errorf("failed to delete existing node by path %s: %v", fullPath, err)
		}
	}

	fmt.Print("description>> ")
	description, err := c.consoleReader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read description: %v", err)
	}

	newNode := storage.NewStorageNode(name)
	newNode.SetDescription(strings.TrimSpace(description))

	fmt.Print("tags>> ")
	tagsLine, err := c.consoleReader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read tags: %v", err)
	}
	if tagsLine != "" {
		newNode.AddTags(c.getTokens(tagsLine)...)
	}

	for {
		fmt.Print("data>> ")
		data, err := c.consoleReader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read data: %v", err)
		}

		data = strings.TrimSpace(data)
		if data == "" {
			break
		}

		startValueIndex := strings.Index(data, " ")
		if startValueIndex == -1 {
			fmt.Println("Value must be entered <key> <value> or only <key> for remove value")
			continue
		}

		key := data[:startValueIndex]
		value := data[startValueIndex+1:]

		if key == "" {
			fmt.Println("key cannot be empty")
			break
		}

		newNode.Data().Set(key, value)
	}

	if err = str.PutNode(targetPath, newNode); err != nil {
		return fmt.Errorf("failed to put node '%s' by path '%s': %v", name, targetPath, err)
	}

	logger.Infof("", "Node '%s' created at '%s'\n", name, targetPath)
	fmt.Printf("Node '%s' created at '%s'\n", name, targetPath)
	return nil
}

func (c *CommandProcessorHandler) handleDeleteNode(path string) error {
	str := c.services.Storage
	targetPath := c.createPath(path)
	if targetPath == "" || targetPath == c.positionSeparator || targetPath == c.rootName {
		return fmt.Errorf("path is empty")
	}

	if !str.HasNode(targetPath) {
		return fmt.Errorf("node at '%s' does not exist", targetPath)
	}

	fmt.Printf("Are you sure you want to delete '%s'? (Y/N): ", targetPath)
	res, err := c.consoleReader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read delete confirmation: %v", err)
	}
	res = strings.TrimSpace(res)
	if res != "Y" && res != "y" {
		fmt.Println("Deletion canceled")
		return nil
	}

	if err = str.WipeNode(targetPath); err != nil {
		return fmt.Errorf("failed to delete node by path '%s': %v", targetPath, err)
	}

	if targetPath == c.position || strings.HasPrefix(c.position, targetPath+c.positionSeparator) {
		parts := strings.Split(targetPath, c.positionSeparator)
		if len(parts) > 1 {
			c.position = strings.Join(parts[:len(parts)-1], c.positionSeparator)
		} else {
			c.position = ""
		}
	}

	return nil
}

func (c *CommandProcessorHandler) handleMoveNode(sourcePath, destinationPath string) error {
	str := c.services.Storage
	sourcePath = c.createPath(sourcePath)
	if sourcePath == "" {
		return fmt.Errorf("current position is root, cannot move root node")
	}
	destinationPath = c.createPath(destinationPath)

	sourceParts := strings.Split(sourcePath, c.positionSeparator)
	sourceName := sourceParts[len(sourceParts)-1]

	if destinationPath == sourcePath || strings.HasPrefix(destinationPath, sourcePath+c.positionSeparator) {
		return fmt.Errorf("cannot move '%s' into itself or its descendant", sourcePath)
	}

	parentTargetPath := ""
	if strings.Contains(destinationPath, c.positionSeparator) {
		parentTargetPath = destinationPath[:strings.LastIndex(destinationPath, c.positionSeparator)]
	}
	if parentTargetPath != "" && !str.HasNode(parentTargetPath) {
		return fmt.Errorf("destination parent path '%s' does not exist", parentTargetPath)
	}

	node, err := str.GetNode(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to get source node '%s': %v", sourcePath, err)
	}

	fullTargetPath := destinationPath
	if fullTargetPath != "" {
		fullTargetPath += c.positionSeparator
	}
	fullTargetPath += sourceName
	if str.HasNode(fullTargetPath) {
		fmt.Printf("Node '%s' already exists at '%s'. Overwrite? (Y/N): ", sourceName, destinationPath)
		res, err := c.consoleReader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read response: %v", err)
		}
		res = strings.TrimSpace(res)
		if res != "Y" && res != "y" {
			fmt.Println("Move canceled")
			return nil
		}

		if err = str.DeleteNode(fullTargetPath); err != nil {
			return fmt.Errorf("failed to delete existing node at '%s': %v", fullTargetPath, err)
		}
	}

	if err = str.DeleteNode(sourcePath); err != nil {
		return fmt.Errorf(fmt.Sprintf("failed to delete source node '%s': %v", sourcePath, err))
	}
	if err = str.PutNode(destinationPath, node); err != nil {
		return fmt.Errorf("failed to put node at '%s': %v", destinationPath, err)
	}

	if sourcePath == c.position {
		c.position = fullTargetPath
	}

	logger.Infof("", "Node '%s' moved from '%s' to '%s'\n", sourceName, sourcePath, destinationPath)
	fmt.Printf("Node '%s' moved from '%s' to '%s'\n", sourceName, sourcePath, destinationPath)
	return nil
}

func (c *CommandProcessorHandler) handleChangeNode(path string) error {
	str := c.services.Storage
	targetPath := c.createPath(path)
	if targetPath == "" || targetPath == c.positionSeparator || targetPath == c.rootName {
		return fmt.Errorf("path is empty")
	}

	node, err := str.GetNode(targetPath)
	if err != nil || node == nil {
		return fmt.Errorf("failed to get source node by path '%s': %v", targetPath, err)
	}

	for {
		fmt.Print("name>> ")
		name, err := c.consoleReader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read name: %v", err)
		}
		name = strings.TrimSpace(name)
		if name != "" {
			if strings.ContainsAny(name, c.positionSeparator+"\"") {
				fmt.Println("invalid character in node name")
				continue
			}
			err = node.SetName(name)
			if err != nil {
				fmt.Printf("failed to set name in node '%s': %v\n", targetPath, err)
			}
			break
		} else {
			break
		}
	}

	fmt.Print("description>> ")
	line, _, err := c.consoleReader.ReadLine()
	if err != nil {
		return fmt.Errorf("failed to read description: %v", err)
	}
	description := string(line)
	if description != "" {
		node.SetDescription(description)
	}

	nodeData := node.Data()
	for {
		fmt.Print("data>> ")
		data, err := c.consoleReader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read data: %v", err)
		}

		data = strings.TrimSpace(data)
		if data == "" {
			break
		}

		tokens := strings.Fields(data)
		if len(tokens) == 0 {
			break
		}

		key := tokens[0]
		if key == "" {
			break
		}

		if len(tokens) == 1 {
			if nodeData.Has(key) {
				nodeData.Remove(key)
			}
		} else {
			nodeData.Set(key, strings.Join(tokens[1:], " "))
		}
	}

	return nil
}

func (c *CommandProcessorHandler) handleChangeData(path, key, value string) error {
	str := c.services.Storage

	path = c.createPath(path)
	nodeData, err := str.GetData(path)
	if err != nil || nodeData == nil {
		return fmt.Errorf("node with data does not exist by path '%s': %v", path, err)
	}

	if key != "" {
		if value == "" {
			nodeData.Remove(key)
		} else {
			nodeData.Set(key, value)
		}
		return nil
	}

	for {
		fmt.Print("data>> ")
		data, err := c.consoleReader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read data: %v", err)
		}

		data = strings.TrimSpace(data)
		if data == "" {
			break
		}

		tokens := strings.Fields(data)
		if len(tokens) == 0 {
			break
		}

		key = tokens[0]
		if key == "" {
			break
		}

		if len(tokens) == 1 {
			if nodeData.Has(key) {
				nodeData.Remove(key)
			}
		} else {
			nodeData.Set(key, strings.Join(tokens[1:], " "))
		}
	}
	return nil
}

func (c *CommandProcessorHandler) handleAddNodeTags(path string, tags []string) error {
	if len(tags) == 0 {
		return fmt.Errorf("tags is empty")
	}

	for _, tag := range tags {
		fmt.Printf("|%s|%d\n", tag, len(tag))
	}

	path = c.createPath(path)
	node, err := c.services.Storage.GetNode(path)
	if err != nil || node == nil {
		return fmt.Errorf("node does not exist by path '%s'", path)
	}
	node.AddTags(tags...)

	return nil
}

func (c *CommandProcessorHandler) handleRemoveNodeTags(path string, tags []string) error {
	if len(tags) == 0 {
		return fmt.Errorf("tags is empty")
	}

	path = c.createPath(path)
	node, err := c.services.Storage.GetNode(path)
	if err != nil || node == nil {
		return fmt.Errorf("node does not exist by path '%s'", path)
	}
	node.RemoveTags(tags...)

	return nil
}

func (c *CommandProcessorHandler) handleShowNodesByTags(tags []string) error {
	if len(tags) == 0 {
		return fmt.Errorf("tags is empty")
	}

	str := c.services.Storage
	err := str.WalkNodes(func(node *storage.Node) {
		if node.HasTags(tags...) {
			c.showNode(node)
		}
	})
	if err != nil {
		return fmt.Errorf("failed to walk nodes: %v", err)
	}

	return nil
}

func (c *CommandProcessorHandler) handleSave(path string) error {
	if path == "" {
		path = c.config.Storage.Path()
	}
	return c.services.Storage.SaveStorage(path)
}

func (c *CommandProcessorHandler) getTokens(data string) []string {
	data = strings.TrimSpace(strings.Trim(data, "\""))
	if data == "" {
		return nil
	}

	res := make([]string, 0, 2)
	start := 0
	inQuotes := false

	for i := 0; i < len(data); i++ {
		switch data[i] {
		case '"':
			if inQuotes {
				if i > start {
					res = append(res, strings.TrimSpace(data[start:i]))
				}
				start = i + 1
				inQuotes = false
			} else if i == 0 || c.isSpace(data[i-1]) {
				inQuotes = true
				start = i + 1
			}
		case ' ', '\t', '\n':
			if !inQuotes {
				if i > start {
					res = append(res, data[start:i])
				}
				start = i + 1
			}
		}
	}

	if start < len(data) {
		res = append(res, strings.TrimSpace(data[start:]))
	}

	return res
}

func (c *CommandProcessorHandler) isSpace(b byte) bool {
	return b == ' ' || b == '\t' || b == '\n'
}

func (c *CommandProcessorHandler) createPath(path string) string {
	targetPath := c.position
	if path != "" {
		if strings.HasPrefix(path, "/") {
			targetPath = strings.TrimPrefix(path, "/")
		} else if targetPath == "" {
			targetPath = path
		} else {
			targetPath += c.positionSeparator + path
		}
	}
	return strings.Trim(targetPath, c.positionSeparator)
}
