import re

# example: 
# command: 'shoppingList: [action:add,remove,delete] {what} [to,from] {when} * shopping list"'
# translation to re:
# echo "shoppingList: [action:add,remove,delete] {what} [to,from] {when}* shopping list" | sed -r -f poc.sed
# the sharp command is not implemented yet

# usage:
p = re.compile(r'^(?P<action>add|remove|delete) (?P<what>.*) (to|from) (?P<when>.*).* shopping list$')
m = p.search("add potatoes to tomorrow's shopping list")
print(m.group('action'))
print(m.group('what'))
