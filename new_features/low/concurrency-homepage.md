## Future Enhancement: Concurrent Homepage Assignment

### Current Implementation

- A unique partial MongoDB index on `is_homepage` ensures that at most one page can have `is_homepage: true`.
- Assigning a new homepage first removes the designation from the existing homepage and then assigns it to the selected page.
- This works for normal sequential requests.

### Limitation

If two admins assign different pages as homepage simultaneously, both requests may attempt to update the homepage designation at the same time.

- One request may fail with a MongoDB duplicate-key error.
- The unique index prevents two pages from successfully retaining `is_homepage: true`.
- The current two-step reassignment is not atomic.

### Future Improvement

Implement transaction-based homepage reassignment:

1. Start a MongoDB transaction.
2. Remove the homepage designation from the existing page.
3. Assign the designation to the newly selected page.
4. Commit the transaction.
5. Roll back if an operation fails.

Keep the unique partial index as an additional safeguard.

### Requirements

- MongoDB must run as a replica set or through a sharded cluster to support transactions.
- A single-node replica set is sufficient; multiple servers are not required.
- Handle transient transaction errors and retry when appropriate.
- Add automated concurrent-request tests to verify the behaviour.

### Current Status

**Deferred.** The existing implementation is sufficient for current application testing. Revisit this enhancement if concurrent admin operations become a realistic concern.