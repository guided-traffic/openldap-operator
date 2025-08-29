# Automatic Home Directory Configuration

## Overview

The OpenLDAP Operator now automatically sets a home directory for POSIX users when none is specified. This ensures that all POSIX accounts have a valid home directory, which is required by the LDAP schema.

## How it works

When creating or updating a `LDAPUser`, if the `homeDirectory` field is not specified or is empty, the controller will automatically set it to `/home/<username>`.

## Examples

### User with explicit home directory
```yaml
apiVersion: openldap.guided-traffic.com/v1
kind: LDAPUser
metadata:
  name: user-with-custom-home
spec:
  username: customuser
  homeDirectory: /custom/path/customuser  # Explicitly set
  # ... other fields
```

Result: The user will have `/custom/path/customuser` as home directory.

### User without home directory (automatic)
```yaml
apiVersion: openldap.guided-traffic.com/v1
kind: LDAPUser
metadata:
  name: user-with-auto-home
spec:
  username: autouser
  # homeDirectory is not specified
  # ... other fields
```

Result: The user will automatically get `/home/autouser` as home directory.

### User with empty home directory (automatic)
```yaml
apiVersion: openldap.guided-traffic.com/v1
kind: LDAPUser
metadata:
  name: user-with-empty-home
spec:
  username: emptyuser
  homeDirectory: ""  # Explicitly empty
  # ... other fields
```

Result: The user will automatically get `/home/emptyuser` as home directory.

## Status Tracking

The actual home directory that was set in LDAP is tracked in the status field:

```yaml
status:
  actualHomeDirectory: /home/username
  # ... other status fields
```

This allows you to see what home directory was actually configured, whether it was explicitly set or automatically generated.

## Benefits

1. **POSIX Compliance**: Ensures all POSIX accounts have a valid home directory
2. **Convenience**: No need to manually specify home directories for standard users
3. **Flexibility**: Still allows custom home directories when needed
4. **Transparency**: Shows the actual configured value in the status
