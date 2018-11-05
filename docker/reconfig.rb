# Ruby script to convert a Jenkins-generated config.json into a proper nested JSON
# Expects one parameter - the file name of the json config to modify
#   e.g. $ ruby reconfig.rb config.json
#
# For example, a Jenkins config.json may look like this:
# [
#     {
#         "key": "accounts_test_region",
#         "value": "us-east-1"
#     },
#     {
#         "key": "accounts_test_key",
#         "value": "KEY"
#     },
#     {
#         "key": "accounts_test_secret",
#         "value": "SECRET"
#     },
#     {
#         "key": "port",
#         "value": "8080"
#     }
# ]
#
# After processing (using _ as the separator), the config will be printed as:
# {
#   "accounts": {
#     "test": {
#       "region": "us-east-1",
#       "key": "KEY",
#       "secret": "SECRET"
#     }
#   },
#   "port": "8080"
# }

require 'json'

# separator character
SEP = '_'.freeze

if ARGV.length.zero?
  puts 'No config file specified!'
  exit 1
end

CONFIG = ARGV.first
config = JSON.parse(File.read(CONFIG))

if config.is_a?(Array)
  # create auto-vivifying hash
  newconfig = Hash.new { |h, k| h[k] = Hash.new(&h.default_proc) }

  # process all keys and create nested hashes based on the separator
  config.each do |i|
    keys = i['key'].split(SEP)
    keys[0...-1].inject(newconfig) do |acc, h|
      acc.public_send(:[], h)
    end.public_send(:[]=, keys.last, i['value'])
  end

  puts JSON.pretty_generate(newconfig)
else
  puts "Cannot process config file #{CONFIG}, aborting!"
  exit 1
end
