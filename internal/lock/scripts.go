package lock

const acquireScript = `
local ttl = tonumber(ARGV[1])
local token = ARGV[2]

for i = 1, #KEYS do
	if redis.call("EXISTS", KEYS[i]) == 1 then
		return 0
	end
end

for i = 1, #KEYS do
	redis.call(
		"SET",
		KEYS[i],
		token,
		"EX",
		ttl
	)
end

return 1
`

const releaseScript = `
local token = ARGV[1]

for i = 1, #KEYS do
	if redis.call("GET", KEYS[i]) ~= token then
		return 0
	end
end

for i = 1, #KEYS do
	redis.call("DEL", KEYS[i])
end

return 1
`

const confirmScript = `
local token = ARGV[1]
local ttl = tonumber(ARGV[2])

for i = 1, #KEYS do
	if redis.call("GET", KEYS[i]) ~= token then
		return 0
	end
end

for i = 1, #KEYS do
	redis.call(
		"SET",
		KEYS[i],
		"BOOKED",
		"EX",
		ttl
	)
end

return 1
`

const refreshScript = `
local token = ARGV[1]
local ttl = tonumber(ARGV[2])

for i = 1, #KEYS do
	if redis.call("GET", KEYS[i]) ~= token then
		return 0
	end
end

for i = 1, #KEYS do
	redis.call(
		"EXPIRE",
		KEYS[i],
		ttl
	)
end

return 1
`
