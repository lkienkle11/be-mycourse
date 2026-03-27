# Auth Module

## Business Logic

- Register with unique email and bcrypt-hashed password.
- Login returns short-lived access token and long-lived refresh token.

## Constraints

- Duplicate email is rejected.
- Only active users can login.

## Transaction Notes

- Registration should run inside a transaction if roles/metadata are inserted together.
