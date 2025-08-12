-- Update the test alarm to have a condition that will always be true
UPDATE context_alarms 
SET condition_data = '{"condition": "current_time_exists", "description": "Check if the current time exists (always true test)", "prompt": "Check if the current time exists. This should always return true since time always exists. Return JSON format with condition_met: true and current time information."}',
    description = 'Test alarm that checks if current time exists (always true for testing Twilio)'
WHERE id = 'test_weather_now';

-- Show the updated alarm
SELECT id, title, description, condition_data, context_data FROM context_alarms WHERE id = 'test_weather_now';
