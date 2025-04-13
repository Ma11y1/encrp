**Actual version: 1.0**

# Storage Application

This application allows interaction with a storage system via a command-line interface. Upon launching the application, you need to provide:

- **Access password** for the storage
- **Path** to the storage

## Commands

**General:**

1. If the path starts with `/`, it is absolute; otherwise it is relative to the current position.
2. If you need to specify a value that contains a space, you must wrap it in quotes `"`.

**The following commands are supported:**

1. **`info`**  
   Displays information about the current storage.

2. **`to <src>`**  
   Navigates to the specified path.

3. **`ls`** or **`list`**  
   Lists all child elements in the current storage location.

4. **`n <src>`** or **`new <src>`**  
   Creates a new element. If a path is provided, the element is created at the specified path, and any missing parent elements along the path are also created.

5. **`rm <src>`** or **`remove <src>`**  
   Deletes the element at the specified path. 

6. **`sh <src>`** or **`show <src>`**  
   Displays the data of the element. If a path is provided, the element at that path is shown; otherwise, the current element is displayed.

7. **`shw <word>`** or **`showword <word>`**  
   Lists all elements in the storage whose name or description contains the specified word.

8. **`dt <src> <key> <value>`** or **`data <src> <key> <value>`**  
   Modifies the data of an element.
    - If no parameters are provided, the current element is modified.
    - If `<src>`, `<key>`, and `<value>` are specified, the data for `<key>` in the element at `<src>` is updated to `<value>`.
    - If only `<src>` and `<key>` are provided, the data for the specified `<key>` is deleted.

9. **`ch <src>`** or **`change <src>`**  
   Modifies the element at the specified path.

10. **`mv <src> <dst>`** or **`move <src> <dst>`**  
    Moves an element.
     - If only `<src>` is provided, the element at the current position is moved.
     - If both `<src>` and `<dst>` are provided, the element at `<src>` is moved to `<dst>`.

11. **`addtag <src> <tag>`**
    Adds a new tag to the element.
     - If only `<tag>` is specified, the tag is added to the current element.
     - If `<stc>` and `<tag>` are specified, the tag is added to the element at the specified path.

12. **`addtags <tags...>`**
    Adds several new tags to current element.

13. **`rmtag <src> <tag>`** or **`removetag <src> <tag>`**
    Removes the specified tag from the element.
     - If only `<tag>` is specified, the tag is removed on the current element.
     - If `<stc>` and `<tag>` are specified, the tag is removed on the element at the specified path.

14. **`rmtags <tags...>`** or **`removetags <tags...>`**
    Removes multiple specified tags from the current element.

15. **`shtags <tags...>`** or **`showtags <tags...>`**
    Displays all elements containing the specified tags.

16. **`save <src>`**
    Saves the current state of the repository. If a path is specified, the repository will be saved to that path.

17. **`e`** or **`exit`**  
    Closes the application without saving changes.