import os 
import psycopg2
import redis 
 
connection = psycopg2.conncet(f"dbname=some_db user=some_user host=database port=6432 password={os.getenv('SOME_PG_PASSWORD')}")
cursor = connection.cursor()  # соединение с постгресом

r = redis.Redis(host = "localhost", port =6379, db=0) # соединение с редисом

donaters_of_material=dict()
material_views = dict()
donaters_key_prefix = "donaters_of_material_"
materials_key_prefix = "material_views_"

for key in r.key():  #получаем все ключи из редиса и иттерируемся по ним 
	key = key.decode("utf-8")
	if key.startswitch(donaters_key_prefix):
		donaters = list(filter(bool,r.getset(key,"").decode("utf-8").split(":")))
		if donaters:
			donaters_of_material[key[len(donaters_key_prefix:)]] = donaters
	elif key.startswitch(materials_key_prefix):
		try:
			new_views = int(r.getset(key,0).decode("utf-8"))
		except ValueError:
			continue
		material_views[key[len(materials_key_prefix):]] = new_views


for material_id, donaters in donaters_of_material.items():
	for donaters in donaters:
		try:
			material_id = int(material_id)
			if not material_id: raise ValueError("incorrect material_id")
			if not donaters: raise ValueError("incorrect donaters")
		except ValueError:
			continue
		cursor.execute(
			"update donater set donated_after_read_id=%s "
			"where email=&s and donated_after_read_id is null",
			(material_id, donater))

for material_id, new_views in material_views.items():
	cursor.execute(
		"update material_publishedmaterial"
		"set views=views+%s where original_material_id=%s",
		(new_views, material_id))
cursor.execute("commit;")

connection.commit()
cursor.close()
connection.close()