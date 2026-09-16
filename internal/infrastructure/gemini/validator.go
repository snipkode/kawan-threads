package gemini

import (
	"errors"
	"fmt"

	"kawan-threads/internal/domain/entity"
	"kawan-threads/internal/domain/port"
)

// validHookTypes is the set of accepted HookType values.
var validHookTypes = map[entity.HookType]bool{
	entity.HookQuestion:       true,
	entity.HookRelatable:      true,
	entity.HookCuriosity:      true,
	entity.HookStory:          true,
	entity.HookMyth:           true,
	entity.HookProblem:        true,
	entity.HookSurprisingFact: true,
	entity.HookHotTake:        true,
}

// ValidateGeneratedContent checks that a GeneratedContent value meets all
// structural requirements. Returns a descriptive error on the first failure.
func ValidateGeneratedContent(gc port.GeneratedContent) error {
	var errs []error

	if gc.Hook == "" {
		errs = append(errs, errors.New("hook is empty"))
	}
	if gc.Body == "" {
		errs = append(errs, errors.New("body is empty"))
	}
	if gc.CTA == "" {
		errs = append(errs, errors.New("cta is empty"))
	}
	if gc.ConversationQuestion == "" {
		errs = append(errs, errors.New("conversationQuestion is empty"))
	}
	if len(gc.HookVariants) == 0 {
		errs = append(errs, errors.New("hookVariants must contain at least 1 variant"))
	}
	if !validHookTypes[gc.HookType] && gc.HookType != "" {
		errs = append(errs, fmt.Errorf("invalid hookType: %q", gc.HookType))
	}
	if err := validateScore("quality.hook", gc.Quality.Hook); err != nil {
		errs = append(errs, err)
	}
	if err := validateScore("quality.conversation", gc.Quality.Conversation); err != nil {
		errs = append(errs, err)
	}
	if err := validateScore("quality.readability", gc.Quality.Readability); err != nil {
		errs = append(errs, err)
	}
	if err := validateScore("quality.originality", gc.Quality.Originality); err != nil {
		errs = append(errs, err)
	}
	if err := validateScore("quality.relevance", gc.Quality.Relevance); err != nil {
		errs = append(errs, err)
	}
	if err := validateScore("quality.shareability", gc.Quality.Shareability); err != nil {
		errs = append(errs, err)
	}
	if err := validateScore("conversationScore", gc.ConversationScore); err != nil {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		msgs := make([]string, len(errs))
		for i, e := range errs {
			msgs[i] = e.Error()
		}
		return fmt.Errorf("generated content validation failed: %s", joinStrings(msgs, "; "))
	}
	return nil
}

func validateScore(field string, score int) error {
	if score < 0 || score > 100 {
		return fmt.Errorf("%s must be 0–100, got %d", field, score)
	}
	return nil
}

func joinStrings(ss []string, sep string) string {
	if len(ss) == 0 {
		return ""
	}
	result := ss[0]
	for _, s := range ss[1:] {
		result += sep + s
	}
	return result
}
