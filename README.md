**Actual version: 1.0**

# Storage Application

This application allows interaction with a storage system via a command-line interface. Upon launching the application, you need to provide:

- **Access password** for the storage
- **Path** to the storage

## Commands

The following commands are supported:

1. **`info`**  
   Displays information about the current storage.

2. **`to <src>`**  
   Navigates to the specified path. If the path starts with `/`, it is treated as an absolute path; otherwise, navigation is relative to the current position.

3. **`ls`** or **`list`**  
   Lists all child elements in the current storage location.

4. **`n <src>`** or **`new <src>`**  
   Creates a new element. If a path is provided, the element is created at the specified path, and any missing parent elements along the path are also created. If the path starts with `/`, it is absolute; otherwise, it is relative to the current position.

5. **`rm <src>`** or **`remove <src>`**  
   Deletes the element at the specified path. If the path starts with `/`, it is absolute; otherwise, it is relative to the current position.

6. **`sh <src>`** or **`show <src>`**  
   Displays the data of the element. If a path is provided, the element at that path is shown; otherwise, the current element is displayed. If the path starts with `/`, it is absolute; otherwise, it is relative to the current position.

7. **`shw <word>`** or **`showword <word>`**  
   Lists all elements in the storage whose name or description contains the specified word.

8. **`dt <src> <key> <value>`** or **`data <src> <key> <value>`**  
   Modifies the data of an element.
    - If no parameters are provided, the current element is modified.
    - If `<src>`, `<key>`, and `<value>` are specified, the data for `<key>` in the element at `<src>` is updated to `<value>`.
    - If only `<src>` and `<key>` are provided, the data for the specified `<key>` is deleted.

9. **`ch <src>`** or **`change <src>`**  
   Modifies the element at the specified path. If the path starts with `/`, it is absolute; otherwise, it is relative to the current position.

10. **`mv <src> <dst>`** or **`move <src> <dst>`**  
    Moves an element.
    - If only `<src>` is provided, the element at the current position is moved.
    - If both `<src>` and `<dst>` are provided, the element at `<src>` is moved to `<dst>`.
    - If the path starts with `/`, it is absolute; otherwise, it is relative to the current position.

11. **`save <src>`**
    Saves the current state of the repository.
     - If a path is specified, the repository will be saved to that path.
     - If the path begins with `/`, it is absolute; otherwise, it is relative to the current position.

12. **`e`** or **`exit`**  
    Closes the application without saving changes.