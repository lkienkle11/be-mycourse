package infra

import (
	"context"

	"golang.org/x/sync/errgroup"
	"gorm.io/gorm"

	"mycourse-io-be/internal/course/domain"
	"mycourse-io-be/internal/shared/constants"
)

func (r *GormRepository) batchHydrateSubLessons(ctx context.Context, db *gorm.DB, rows []subLessonRow) (map[string]domain.SubLesson, error) {
	out, videoIDs, textIDs, quizIDs := seedSubLessonContentMap(rows)
	if len(rows) == 0 {
		return out, nil
	}
	group, gctx := errgroup.WithContext(ctx)
	if len(videoIDs) > 0 {
		group.Go(func() error { return r.hydrateSubLessonVideos(gctx, parallelReadDB(db), out, videoIDs) })
	}
	if len(textIDs) > 0 {
		group.Go(func() error { return r.hydrateSubLessonTexts(gctx, parallelReadDB(db), out, textIDs) })
	}
	if len(quizIDs) > 0 {
		group.Go(func() error { return r.hydrateSubLessonQuizzes(gctx, parallelReadDB(db), out, quizIDs) })
	}
	if err := group.Wait(); err != nil {
		return nil, err
	}
	return out, nil
}

func seedSubLessonContentMap(rows []subLessonRow) (
	out map[string]domain.SubLesson,
	videoIDs, textIDs, quizIDs []string,
) {
	out = make(map[string]domain.SubLesson, len(rows))
	for _, row := range rows {
		out[row.ID] = toSubLesson(&row)
		switch row.Kind {
		case domain.SubLessonKindVideo:
			videoIDs = append(videoIDs, row.ID)
		case domain.SubLessonKindText:
			textIDs = append(textIDs, row.ID)
		case domain.SubLessonKindQuiz:
			quizIDs = append(quizIDs, row.ID)
		}
	}
	return out, videoIDs, textIDs, quizIDs
}

func (r *GormRepository) hydrateSubLessonVideos(
	ctx context.Context,
	db *gorm.DB,
	out map[string]domain.SubLesson,
	videoIDs []string,
) error {
	var videos []subLessonVideoRow
	if err := db.WithContext(ctx).Where("sub_lesson_id IN ?", videoIDs).Find(&videos).Error; err != nil {
		return err
	}
	fileIDs := make([]string, 0, len(videos))
	videoBySub := make(map[string]subLessonVideoRow, len(videos))
	for _, video := range videos {
		videoBySub[video.SubLessonID] = video
		fileIDs = append(fileIDs, video.MediaFileID)
	}
	urlMap, err := r.batchMediaURLMap(ctx, db, fileIDs)
	if err != nil {
		return err
	}
	for subID, video := range videoBySub {
		sub := out[subID]
		sub.Video = &domain.VideoContent{
			MediaFileID: video.MediaFileID,
			MediaURL:    urlMap[video.MediaFileID],
		}
		out[subID] = sub
	}
	return nil
}

func (r *GormRepository) hydrateSubLessonTexts(
	ctx context.Context,
	db *gorm.DB,
	out map[string]domain.SubLesson,
	textIDs []string,
) error {
	var texts []subLessonTextRow
	if err := db.WithContext(ctx).Where("sub_lesson_id IN ?", textIDs).Find(&texts).Error; err != nil {
		return err
	}
	for _, text := range texts {
		sub := out[text.SubLessonID]
		sub.Text = &domain.TextContent{ContentDelta: text.ContentDelta}
		out[text.SubLessonID] = sub
	}
	return nil
}

func (r *GormRepository) hydrateSubLessonQuizzes(
	ctx context.Context,
	db *gorm.DB,
	out map[string]domain.SubLesson,
	quizIDs []string,
) error {
	var quizzes []subLessonQuizRow
	if err := db.WithContext(ctx).Where("sub_lesson_id IN ?", quizIDs).Find(&quizzes).Error; err != nil {
		return err
	}
	var options []subLessonQuizOptionRow
	if err := db.WithContext(ctx).
		Where("sub_lesson_id IN ?", quizIDs).
		Order("order_index ASC").
		Find(&options).Error; err != nil {
		return err
	}
	optionsBySub := make(map[string][]subLessonQuizOptionRow, len(quizIDs))
	for _, option := range options {
		optionsBySub[option.SubLessonID] = append(optionsBySub[option.SubLessonID], option)
	}
	for _, quiz := range quizzes {
		sub := out[quiz.SubLessonID]
		sub.Quiz = &domain.QuizContent{
			Prompt:        quiz.Prompt,
			AllowMultiple: quiz.AllowMultiple,
			Options:       mapQuizOptions(optionsBySub[quiz.SubLessonID]),
		}
		out[quiz.SubLessonID] = sub
	}
	return nil
}

func (r *GormRepository) batchMediaURLMap(ctx context.Context, db *gorm.DB, fileIDs []string) (map[string]string, error) {
	out := make(map[string]string)
	if len(fileIDs) == 0 {
		return out, nil
	}
	unique := make([]string, 0, len(fileIDs))
	seen := make(map[string]struct{}, len(fileIDs))
	for _, id := range fileIDs {
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		unique = append(unique, id)
	}
	if len(unique) == 0 {
		return out, nil
	}
	var rows []mediaInfoRow
	if err := db.WithContext(ctx).
		Table(constants.TableMediaFiles).
		Select("id, url").
		Where("id IN ? AND deleted_at IS NULL", unique).
		Find(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		out[row.ID] = row.URL
	}
	return out, nil
}
