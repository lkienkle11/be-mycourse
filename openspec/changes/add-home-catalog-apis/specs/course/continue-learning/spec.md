## Purpose

Lets an authenticated learner see the courses they are actively studying, ranked by how recently they engaged with them, so the Home page can surface exactly where to resume — including courses just enrolled but not yet started.

## ADDED Requirements

### Requirement: Continue-learning list requires authentication
The system SHALL require a valid authenticated session to access the continue-learning list, consistent with other learner endpoints.

#### Scenario: Anonymous request rejected
- **WHEN** a client calls the continue-learning endpoint without a valid session
- **THEN** the system returns the same unauthenticated-rejection response as other `learner-courses` endpoints

### Requirement: Only the caller's own enrollments are returned
The system SHALL return only courses the authenticated caller is enrolled in.

#### Scenario: Learner sees only their own courses
- **WHEN** learner A is enrolled in course X and learner B is enrolled in course Y, and neither is enrolled in the other's course
- **THEN** a request authenticated as learner A returns course X and does not return course Y

### Requirement: List is ordered by most recent learning activity, falling back to enrollment time
The system SHALL order the list by the most recent lesson-progress interaction across the learner's enrollments, using the enrollment's creation time as the activity timestamp when no lesson progress exists yet.

#### Scenario: Revisiting a completed course promotes it to the top
- **WHEN** a learner has completed course X and later revisits any lesson in X, and has not interacted with course Y more recently
- **THEN** the system ranks course X above course Y

#### Scenario: Freshly enrolled, unstarted course still ranks by enrollment recency
- **WHEN** a learner enrolls in course Z and has not opened any lesson in it yet
- **THEN** the system ranks course Z using its enrollment time, so it is not omitted or pushed below courses with older activity purely for having no lesson-progress rows yet

### Requirement: Each item includes a sub-lesson completion progress summary
The system SHALL include, for each returned course, the count of completed sub-lessons and the total number of sub-lessons in the learner's current course version, counting every sub-lesson kind (video, quiz, and text) rather than video-only, since a course's outline legitimately mixes content kinds under one course.

#### Scenario: Progress counts reflect actual completion across all content kinds
- **WHEN** a course has N sub-lessons across any mix of video, quiz, and text kinds, and the learner has completed M of them
- **THEN** the response for that course includes both M and N

### Requirement: Result count is bounded by a limit parameter
The system SHALL accept an optional `limit` parameter, applying a default and a maximum when it is omitted or exceeds the maximum.

#### Scenario: Explicit limit respected
- **WHEN** a client requests the continue-learning list with `limit=2`
- **THEN** the system returns at most 2 items

#### Scenario: Omitted limit uses default
- **WHEN** a client requests the continue-learning list without a `limit` parameter
- **THEN** the system returns the default number of items

#### Scenario: Excessive limit is clamped
- **WHEN** a client requests a `limit` above the documented maximum
- **THEN** the system clamps the result to the maximum instead of rejecting the request
