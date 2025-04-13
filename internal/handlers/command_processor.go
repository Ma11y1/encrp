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
	actions           map[string]func([]string) error
}

func NewCommandProcessorHandler(cfg *config.Config, handlers *Container, services *services.Container) *CommandProcessorHandler {
	h := &CommandProcessorHandler{
		config:            cfg,
		handlers:          handlers,
		services:          services,
		consoleReader:     bufio.NewReader(os.Stdin),
		positionSeparator: cfg.Storage.PathSeparator(),
	}
	actions := make(map[string]func([]string) error)
	actions["info"] = h.handleShowInfo
	actions["to"] = h.handleMoveToPath

	actions["ls"] = h.handleShowChildrenList
	actions["list"] = h.handleShowChildrenList

	actions["n"] = h.handleCreateNode
	actions["new"] = h.handleCreateNode

	actions["rm"] = h.handleRemoveNode
	actions["remove"] = h.handleRemoveNode

	actions["sh"] = h.handleShowNode
	actions["show"] = h.handleShowNode

	actions["shw"] = h.handleShowNodesByWord
	actions["showword"] = h.handleShowNodesByWord

	actions["dt"] = h.handleChangeData
	actions["data"] = h.handleChangeData

	actions["ch"] = h.handleChangeNode
	actions["change"] = h.handleChangeNode

	actions["mv"] = h.handleMoveNode
	actions["move"] = h.handleMoveNode

	actions["addtag"] = h.handleAddNodeTag
	actions["addtags"] = h.handleAddNodeTags

	actions["rmtag"] = h.handleRemoveNodeTag
	actions["removetag"] = h.handleRemoveNodeTag

	actions["rmtags"] = h.handleRemoveNodeTags
	actions["removetags"] = h.handleRemoveNodeTags

	actions["shtags"] = h.handleShowNodesByTags
	actions["showtags"] = h.handleShowNodesByTags

	actions["save"] = h.handleSave
	h.actions = actions
	return h
}

func (c *CommandProcessorHandler) Start(ctx context.Context) error {
	st := c.services.Storage.GetStorage()
	if st == nil || st.Data() == nil || st.Data().Name() == "" {
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
			fmt.Println("error reading command: ", err)
		}

		if err = ctx.Err(); err != nil {
			return errors.Wrap(err, "CommandProcessorHandler.Start()", "Termination of execution by context")
		}

		tokens := c.getTokens(command)
		if len(tokens) == 0 {
			fmt.Println("empty command")
			continue
		}

		if tokens[0] == "exit" {
			return nil
		}

		action, exists := c.actions[tokens[0]]
		if !exists {
			logger.Warnf("CommandProcessorHandler.Start()", "Invalid command '%s'", command)
			fmt.Printf("invalid command '%s'\n", command)
			continue
		}

		logger.Infof("CommandProcessorHandler.Start()", "Executing command '%s'", command)

		err = action(tokens[1:])
		if err != nil {
			logger.Warnf("CommandProcessorHandler.Start()", "Error processing command %s: %s", command, err)
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

func (c *CommandProcessorHandler) handleShowInfo([]string) error {
	s := c.services.Storage.GetStorage()
	if s == nil {
		return errors.New("CommandProcessorHandler.handleShowInfo()", "No loaded storage or it is invalid")
	}
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
	return nil
}

func (c *CommandProcessorHandler) handleMoveToPath(tokens []string) error {
	if len(tokens) == 0 {
		fmt.Println("invalid syntax command: 'to <src>'")
		return errors.Newf("CommandProcessorHandler.handleMoveToPath()", "Invalid syntax command: 'to <src>'")
	}

	path := tokens[0]
	if path == "" {
		fmt.Println("path is empty")
		return errors.New("CommandProcessorHandler.handleMoveToPath()", "path is empty")
	}

	if path == "/" {
		c.position = ""
		return nil
	}

	newPosition := c.createPath(path)
	if !c.services.Storage.HasNode(newPosition) {
		fmt.Printf("path '%s' does not exist\n", newPosition)
		return errors.Newf("CommandProcessorHandler.handleMoveToPath()", "path '%s' does not exist", newPosition)
	}
	c.position = newPosition
	return nil
}

func (c *CommandProcessorHandler) handleShowChildrenList([]string) error {
	str := c.services.Storage

	node, err := str.GetNode(c.position)
	if err != nil {
		fmt.Printf("Error getting node by path '%s'\n", c.position)
		return errors.Wrapf(err, "CommandProcessorHandler.handleShowChildrenList()", "Failed to getting node by path '%s': %v", c.position, err)
	}

	children := node.Children().Values()
	if len(children) == 0 {
		fmt.Println("node has no children")
		return errors.New("CommandProcessorHandler.handleShowChildrenList()", "Node has no children")
	}

	fmt.Printf("№  | %-20s | %-19s | %-19s | %-20s | %-7s | %-30s |\n", "Name", "Created", "Modified", "Tags", "Children", "Description")
	for i, child := range children {
		fmt.Printf("%d. | %-20s | %-19s | %-19s | %-20s | %-8d | %-30s |\n",
			i+1,
			child.Name(),
			time.Unix(child.TsCreate(), 0).Format("2006-01-02 15:04:05"),
			time.Unix(child.TsModify(), 0).Format("2006-01-02 15:04:05"),
			strings.Join(child.Tags(), ";"),
			child.Children().Count(),
			child.Description(),
		)
	}

	return nil
}

func (c *CommandProcessorHandler) handleCreateNode(tokens []string) error {
	path := ""
	if len(tokens) > 0 {
		path = tokens[0]
	}

	str := c.services.Storage
	targetPath := c.createPath(path)

	fmt.Print("name>> ")
	name, err := c.consoleReader.ReadString('\n')
	if err != nil {
		fmt.Println("failed to read node name")
		return errors.Wrapf(err, "CommandProcessorHandler.handleCreateNode()", "Failed to read node name: %v", err)
	}
	name = strings.TrimSpace(name)
	if name == "" {
		fmt.Println("name is empty")
		return errors.New("CommandProcessorHandler.handleCreateNode()", "Name is empty")
	}
	if strings.ContainsAny(name, c.positionSeparator+"\"") {
		fmt.Printf("invalid character in node name '%s'\n", name)
		return errors.Newf("CommandProcessorHandler.handleCreateNode()", "Invalid character in node name '%s'", name)
	}

	fullPath := targetPath
	if fullPath != "" {
		fullPath += c.positionSeparator
	}
	fullPath += name

	if str.HasNode(fullPath) {
		fmt.Print("node already exists, overwrite? (Y/N): ")
		res, err := c.consoleReader.ReadString('\n')
		if err != nil {
			fmt.Println("failed to read response to node rewrite question")
			return errors.Wrap(err, "CommandProcessorHandler.handleCreateNode()", "Failed to read response to node rewrite question")
		}
		res = strings.TrimSpace(res)
		if res != "Y" && res != "y" {
			return nil
		}

		if err = str.WipeNode(fullPath); err != nil {
			fmt.Printf("failed to delete existing node by path %s\n", fullPath)
			return errors.Wrapf(err, "CommandProcessorHandler.handleCreateNode()", "Failed to delete existing node by path %s", fullPath)
		}
	}

	fmt.Print("description>> ")
	description, err := c.consoleReader.ReadString('\n')
	if err != nil {
		fmt.Println("failed to read node description")
		return errors.Wrap(err, "CommandProcessorHandler.handleCreateNode()", "Failed to read node description")
	}

	newNode := storage.NewStorageNode(name)
	newNode.SetDescription(strings.TrimSpace(description))

	fmt.Print("tags>> ")
	tagsLine, err := c.consoleReader.ReadString('\n')
	if err != nil {
		fmt.Println("failed to read tags")
		return errors.Wrap(err, "CommandProcessorHandler.handleCreateNode()", "Failed to read tags")
	}
	if tagsLine != "" {
		newNode.AddTags(c.getTokens(tagsLine)...)
	}

	for {
		fmt.Print("data>> ")
		data, err := c.consoleReader.ReadString('\n')
		if err != nil {
			fmt.Println("failed to read data")
			return errors.Wrap(err, "CommandProcessorHandler.handleCreateNode()", "Failed to read data")
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
		fmt.Printf("failed to put node '%s' by path '%s'\n", name, targetPath)
		return errors.Wrapf(err, "CommandProcessorHandler.handleCreateNode()", "Failed to put node '%s' by path '%s'", name, targetPath)
	}

	logger.Infof("CommandProcessorHandler.handleCreateNode()", "Node '%s' created at '%s'\n", name, targetPath)
	fmt.Printf("node '%s' created at '%s'\n", name, targetPath)
	return nil
}

func (c *CommandProcessorHandler) handleRemoveNode(tokens []string) error {
	if len(tokens) == 0 {
		fmt.Println("invalid syntax command: 'remove(rm) <src>'")
		return errors.New("CommandProcessorHandler.handleRemoveNode()", "Invalid syntax command: 'remove(rm) <src>")
	}

	str := c.services.Storage
	targetPath := c.createPath(tokens[0])
	if targetPath == "" || targetPath == c.positionSeparator || targetPath == c.rootName {
		fmt.Println("path is empty'")
		return errors.New("CommandProcessorHandler.handleRemoveNode()", "Path is empty")
	}

	if !str.HasNode(targetPath) {
		fmt.Printf("node at '%s' does not exist\n", targetPath)
		return errors.Newf("CommandProcessorHandler.handleRemoveNode()", "Node at '%s' does not exist", targetPath)
	}

	fmt.Printf("are you sure you want to delete '%s'? (Y/N): ", targetPath)
	res, err := c.consoleReader.ReadString('\n')
	if err != nil {
		fmt.Println("failed to read delete confirmation'")
		return errors.Wrap(err, "CommandProcessorHandler.handleRemoveNode()", "Failed to read delete confirmation")
	}
	res = strings.TrimSpace(res)
	if res != "Y" && res != "y" {
		return nil
	}

	if err = str.WipeNode(targetPath); err != nil {
		fmt.Printf("failed to delete node by path '%s'\n", targetPath)
		return errors.Wrapf(err, "CommandProcessorHandler.handleRemoveNode()", "Failed to delete node by path '%s'", targetPath)
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

func (c *CommandProcessorHandler) handleShowNode(tokens []string) error {
	path := ""
	if len(tokens) > 0 {
		path = tokens[0]
	}

	path = c.createPath(path)
	node, err := c.services.Storage.GetNode(path)
	if err != nil {
		fmt.Printf("failed to get node by path '%s', node not found\n", path)
		return errors.Wrapf(err, "CommandProcessorHandler.handleShowNode()", "Failed to get node by path '%s', node not found", path)
	}

	err = c.showNode(node)
	if err != nil {
		fmt.Printf("failed to show node '%s'\n", node.Name())
		return errors.Wrapf(err, "CommandProcessorHandler.handleShowNode()", "Failed to show node '%s'", node.Name())
	}
	return nil
}

func (c *CommandProcessorHandler) handleShowNodesByWord(tokens []string) error {
	if len(tokens) < 2 {
		fmt.Println("invalid syntax command: 'showword(shw) <word>'")
		return errors.New("CommandProcessorHandler.handleShowNodesByWord()", "Invalid syntax command: 'showword(shw) <word>'")
	}

	word := tokens[0]
	if word == "" {
		fmt.Println("word is empty")
		return errors.New("CommandProcessorHandler.handleShowNodesByWord()", "Word is empty")
	}

	str := c.services.Storage
	err := str.WalkNodes(func(node *storage.Node) {
		if strings.Contains(node.Name(), word) || strings.Contains(node.Description(), word) {
			err := c.showNode(node)
			if err != nil {
				fmt.Printf("failed to show node '%s'\n", node.Name())
			}
		}
	})
	if err != nil {
		fmt.Println("failed to walk nodes")
		return errors.Wrap(err, "CommandProcessorHandler.handleShowNodesByWord()", "Failed to walk nodes")
	}

	return nil
}

func (c *CommandProcessorHandler) showNode(node *storage.Node) error {
	if node == nil {
		fmt.Println("node is nil")
		return errors.New("CommandProcessorHandler.showNode()", "Node is nil")
	}

	fmt.Printf("------------ [%s] ------------\ncreate: %s\nmodify: %s\n",
		node.Name(),
		time.Unix(node.TsCreate(), 0).Format("2006-01-02 15:04:05"),
		time.Unix(node.TsModify(), 0).Format("2006-01-02 15:04:05"),
	)

	tags := node.Tags()
	if len(tags) > 0 {
		fmt.Println("tags: ", strings.Join(tags, ";"))
	}

	if node.Description() != "" {
		fmt.Println("description: ", node.Description())
	}

	fmt.Println("children: ", node.Children().Count())
	nodeChildren := node.Children()
	if nodeChildren.Count() > 0 {
		for i, name := range nodeChildren.Names() {
			child := nodeChildren.Get(name)
			if child == nil {
				continue
			}
			fmt.Printf("\t%d. [%s] [children: %d] [create: %s] [modify: %s] [tags: %s] descr: %s\n",
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
		fmt.Println("data: ")
		nodeData := node.Data()
		for _, key := range dataKeys {
			fmt.Printf("\t%s: %s\n", key, nodeData.Get(key))
		}
	}
	fmt.Println("------------------------")
	return nil
}

func (c *CommandProcessorHandler) handleChangeData(tokens []string) error {
	path, key, value := "", "", ""
	if len(tokens) > 0 {
		path = tokens[0]
	}
	if len(tokens) > 1 {
		key = tokens[1]
	}
	if len(tokens) > 2 {
		value = tokens[2]
	}

	path = c.createPath(path)
	nodeData, err := c.services.Storage.GetData(path)
	if err != nil || nodeData == nil {
		fmt.Printf("node with data does not exist by path '%s'\n", path)
		return errors.Wrapf(err, "CommandProcessorHandler.handleChangeData()", "Node with data does not exist by path '%s'", path)
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
			fmt.Println("failed to read data")
			return errors.Wrap(err, "CommandProcessorHandler.handleChangeData()", "Failed to read data")
		}

		data = strings.TrimSpace(data)
		if data == "" {
			break
		}

		dataTokens := strings.Fields(data)
		if len(dataTokens) == 0 {
			break
		}

		key = dataTokens[0]
		if key == "" {
			break
		}

		if len(dataTokens) == 1 {
			if nodeData.Has(key) {
				nodeData.Remove(key)
			}
		} else {
			nodeData.Set(key, strings.Join(dataTokens[1:], " "))
		}
	}
	return nil
}

func (c *CommandProcessorHandler) handleChangeNode(tokens []string) error {
	if len(tokens) == 0 {
		fmt.Println("Invalid syntax command: 'change(ch) <src>'")
		return errors.New("CommandProcessorHandler.handleChangeNode()", "Invalid syntax command: 'change(ch) <src>")
	}

	path := c.createPath(tokens[0])
	if path == "" || path == c.positionSeparator || path == c.rootName {
		fmt.Println("path is empty")
		return errors.New("CommandProcessorHandler.handleChangeNode()", "Path is empty")
	}

	node, err := c.services.Storage.GetNode(path)
	if err != nil || node == nil {
		fmt.Printf("failed to get source node by path '%s'\n", path)
		return errors.Wrapf(err, "CommandProcessorHandler.handleChangeNode()", "Failed to get source node by path '%s'", path)
	}

	for {
		fmt.Print("name>> ")
		name, err := c.consoleReader.ReadString('\n')
		if err != nil {
			fmt.Println("failed to read name")
			return errors.Wrap(err, "CommandProcessorHandler.handleChangeNode()", "Failed to read name")
		}
		name = strings.TrimSpace(name)
		if name != "" {
			if strings.ContainsAny(name, c.positionSeparator+"\"") {
				fmt.Printf("invalid character in node name '%s'\n", name)
				continue
			}
			err = node.SetName(name)
			if err != nil {
				fmt.Printf("failed to set name in node '%s'\n", path)
				logger.Warnf("CommandProcessorHandler.handleChangeNode()", "Failed to set name in node '%s': %v", path, err)
			}
			break
		} else {
			break
		}
	}

	fmt.Print("description>> ")
	line, _, err := c.consoleReader.ReadLine()
	if err != nil {
		fmt.Println("failed to read description")
		return errors.Wrap(err, "CommandProcessorHandler.handleChangeNode()", "Failed to read description")
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
			fmt.Println("failed to read data")
			return errors.Wrap(err, "CommandProcessorHandler.handleChangeNode()", "Failed to read data")
		}

		data = strings.TrimSpace(data)
		if data == "" {
			break
		}

		dataTokens := strings.Fields(data)
		if len(dataTokens) == 0 {
			break
		}

		key := dataTokens[0]
		if key == "" {
			break
		}

		if len(dataTokens) == 1 {
			if nodeData.Has(key) {
				nodeData.Remove(key)
			}
		} else {
			nodeData.Set(key, strings.Join(dataTokens[1:], " "))
		}
	}

	return nil
}

func (c *CommandProcessorHandler) handleMoveNode(tokens []string) error {
	if len(tokens) == 0 {
		fmt.Println("invalid syntax command: 'move(mv) <src> <dst> or move(mv) <dst>'")
		return errors.New("CommandProcessorHandler.handleMoveNode()", "Invalid syntax command: 'move(mv) <src> <dst> or move(mv) <dst>")
	}
	sourcePath := ""
	destinationPath := tokens[0]
	if len(tokens) > 1 {
		sourcePath = tokens[0]
		destinationPath = tokens[1]
	}

	str := c.services.Storage
	sourcePath = c.createPath(sourcePath)
	if sourcePath == "" {
		fmt.Println("current position is root, cannot move root node")
		return errors.New("CommandProcessorHandler.handleMoveNode()", "Current position is root, cannot move root node")
	}
	destinationPath = c.createPath(destinationPath)

	sourceParts := strings.Split(sourcePath, c.positionSeparator)
	sourceName := sourceParts[len(sourceParts)-1]

	if destinationPath == sourcePath || strings.HasPrefix(destinationPath, sourcePath+c.positionSeparator) {
		fmt.Printf("cannot move '%s' into itself or its descendant\n", sourcePath)
		return errors.Newf("CommandProcessorHandler.handleMoveNode()", "Cannot move '%s' into itself or its descendant", sourcePath)
	}

	parentTargetPath := ""
	if strings.Contains(destinationPath, c.positionSeparator) {
		parentTargetPath = destinationPath[:strings.LastIndex(destinationPath, c.positionSeparator)]
	}
	if parentTargetPath != "" && !str.HasNode(parentTargetPath) {
		fmt.Printf("destination parent path '%s' does not exist\n", parentTargetPath)
		return errors.Newf("CommandProcessorHandler.handleMoveNode()", "Destination parent path '%s' does not exist", parentTargetPath)
	}

	node, err := str.GetNode(sourcePath)
	if err != nil {
		fmt.Printf("failed to get source node '%s'\n", sourcePath)
		return errors.Newf("CommandProcessorHandler.handleMoveNode()", "Failed to get source node '%s'", sourcePath)
	}

	fullTargetPath := destinationPath
	if fullTargetPath != "" {
		fullTargetPath += c.positionSeparator
	}
	fullTargetPath += sourceName
	if str.HasNode(fullTargetPath) {
		fmt.Printf("node '%s' already exists at '%s', overwrite? (Y/N): ", sourceName, destinationPath)
		res, err := c.consoleReader.ReadString('\n')
		if err != nil {
			fmt.Println("failed to read response")
			return errors.Wrap(err, "CommandProcessorHandler.handleMoveNode()", "Failed to read response")
		}
		res = strings.TrimSpace(res)
		if res != "Y" && res != "y" {
			return nil
		}

		if err = str.DeleteNode(fullTargetPath); err != nil {
			fmt.Printf("failed to delete existing node at '%s'\n", fullTargetPath)
			return errors.Wrapf(err, "CommandProcessorHandler.handleMoveNode()", "Failed to delete existing node at '%s'", fullTargetPath)
		}
	}

	if err = str.DeleteNode(sourcePath); err != nil {
		fmt.Printf("failed to delete source node '%s'\n", sourcePath)
		return errors.Wrapf(err, "CommandProcessorHandler.handleMoveNode()", "Failed to delete source node '%s'", sourcePath)
	}

	if err = str.PutNode(destinationPath, node); err != nil {
		fmt.Printf("failed to put node at '%s'\n", destinationPath)
		return errors.Wrapf(err, "CommandProcessorHandler.handleMoveNode()", "Failed to put node at '%s'", destinationPath)
	}

	if sourcePath == c.position {
		c.position = fullTargetPath
	}

	logger.Infof("", "Node '%s' moved from '%s' to '%s'\n", sourceName, sourcePath, destinationPath)
	fmt.Printf("node '%s' moved from '%s' to '%s'\n", sourceName, sourcePath, destinationPath)
	return nil
}

func (c *CommandProcessorHandler) handleAddNodeTag(tokens []string) error {
	if len(tokens) == 0 {
		fmt.Println("invalid syntax command: 'addtag <tag> or addtag <src> <tag>")
		return errors.New("CommandProcessorHandler.handleAddNodeTag()", "Invalid syntax command: 'addtag <tag> or addtag <src> <tag>")
	}

	path, tag := "", tokens[0]
	if len(tokens) > 1 {
		path = tokens[0]
		tag = tokens[1]
	}

	path = c.createPath(path)
	node, err := c.services.Storage.GetNode(path)
	if err != nil || node == nil {
		fmt.Printf("node does not exist by path '%s'\n", path)
		return errors.Wrapf(err, "CommandProcessorHandler.handleAddNodeTag()", "Node does not exist by path '%s'", path)
	}
	node.AddTags(tag)
	return nil
}

func (c *CommandProcessorHandler) handleAddNodeTags(tokens []string) error {
	if len(tokens) == 0 {
		fmt.Println("invalid syntax command: 'addtags <tags...>")
		return errors.New("CommandProcessorHandler.handleAddNodeTags()", "Invalid syntax command: 'addtags <tags...>")
	}

	node, err := c.services.Storage.GetNode(c.position)
	if err != nil || node == nil {
		fmt.Printf("node does not exist by path '%s'\n", c.position)
		return errors.Wrapf(err, "CommandProcessorHandler.handleAddNodeTags()", "Node does not exist by path '%s'", c.position)
	}
	node.AddTags(tokens...)
	return nil
}

func (c *CommandProcessorHandler) handleRemoveNodeTag(tokens []string) error {
	if len(tokens) == 0 {
		fmt.Println("invalid syntax command: 'removetag(rmtag) <tag> or removetag(rmtag) <src> <tag>")
		return errors.New("CommandProcessorHandler.handleRemoveNodeTag()", "Invalid syntax command: 'removetag(rmtag) <tag> or removetag(rmtag) <src> <tag>")
	}

	path, tag := "", tokens[0]
	if len(tokens) > 1 {
		path = tokens[0]
		tag = tokens[1]
	}

	path = c.createPath(path)
	node, err := c.services.Storage.GetNode(path)
	if err != nil || node == nil {
		fmt.Printf("node does not exist by path '%s'\n", path)
		return errors.Wrapf(err, "CommandProcessorHandler.handleRemoveNodeTag()", "Node does not exist by path '%s'", path)
	}
	node.RemoveTags(tag)
	return nil
}

func (c *CommandProcessorHandler) handleRemoveNodeTags(tokens []string) error {
	if len(tokens) == 0 {
		fmt.Println("invalid syntax command: 'removetags(rmtags) <tags...>'")
		return errors.New("CommandProcessorHandler.handleRemoveNodeTags()", "Invalid syntax command: 'removetags(rmtags) <tags...>'")
	}

	node, err := c.services.Storage.GetNode(c.position)
	if err != nil || node == nil {
		fmt.Printf("node does not exist by path '%s'\n", c.position)
		return errors.Wrapf(err, "CommandProcessorHandler.handleRemoveNodeTags()", "Node does not exist by path '%s'", c.position)
	}
	node.RemoveTags(tokens...)
	return nil
}

func (c *CommandProcessorHandler) handleShowNodesByTags(tokens []string) error {
	if len(tokens) == 0 {
		fmt.Println("invalid syntax command: 'showtags(shtags) <tag...>'")
		return errors.New("CommandProcessorHandler.handleShowNodesByTags()", "Invalid syntax command: 'showtags(shtags) <tag...>'")
	}

	if len(tokens) == 0 {
		fmt.Println("tags is empty")
		return errors.New("CommandProcessorHandler.handleShowNodesByTags()", "tags is empty")
	}

	str := c.services.Storage
	err := str.WalkNodes(func(node *storage.Node) {
		if node.HasTags(tokens...) {
			c.showNode(node)
		}
	})
	if err != nil {
		fmt.Println("failed to walk nodes")
		return errors.Wrap(err, "CommandProcessorHandler.handleShowNodesByTags()", "failed to walk nodes")
	}

	return nil
}

func (c *CommandProcessorHandler) handleSave(tokens []string) error {
	path := c.config.Storage.Path()
	if len(tokens) > 0 {
		path = tokens[0]
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
