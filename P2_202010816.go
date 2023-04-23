package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"fmt"
	"io"
	"math"
	"math/rand"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unsafe"
)

type Partition = struct {
	Part_status [1]byte  // Indica si la partición está activa o no
	Part_type   [1]byte  // Valores P o E
	Part_fit    [1]byte  // Indica el Ajuste [B]est, [F]irst o [W]orst
	Part_start  [6]byte  // Indica en qué byte del disco inicia la partición
	Part_size   [6]byte  // Contiene el tamaño total de la partición en bytes
	Part_name   [16]byte // Nombre de la partición
}

type MBR = struct {
	Mbr_tamano         [4]byte   // Tamanio del disco
	Mbr_fecha_creacion [19]byte  // Fecha y hora de creacion del disco
	Mbr_dsk_signature  [4]byte   // Numero random, identifica de forma unica a cada disco
	Dsk_fit            [1]byte   // Ajuste de la particion [B]est, [F]irt o [W]orst
	Mbr_partition_1    Partition // Estructura con información de la partición 1
	Mbr_partition_2    Partition // Estructura con información de la partición 2
	Mbr_partition_3    Partition // Estructura con información de la partición 3
	Mbr_partition_4    Partition // Estructura con información de la partición 4
}

type EBR = struct {
	Part_status [1]byte  // Indica si esta activa o no
	Part_fit    [1]byte  // Indica el Ajuste [B]est, [F]irst o [W]orst
	Part_start  [4]byte  // Indica el byte donde inicia la particion
	Part_size   [4]byte  // Tamanio total de la particion
	Part_next   [4]byte  // Byte en el que está el próximo EBR. -1 si no hay siguiente
	Part_name   [16]byte // Nombre de la particion
}

type SuperBloque = struct {
	S_filesystem_type   [1]byte  // Guarda el número que identifica el sistema de archivos utilizado
	S_inodes_count      [10]byte // Guarda el número total de inodos
	S_blocks_count      [10]byte // Guarda el número total de bloques
	S_free_blocks_count [10]byte // Contiene el número de bloques libres
	S_free_inodes_count [10]byte // Contiene el número de inodos libres
	S_mtime             [19]byte // Última fecha en el que el sistema fue montado
	S_mnt_count         [1]byte  // Indica cuantas veces se ha montado el sistema
	S_magic             [6]byte  // Valor que identifica al sistema de archivos, tendrá el valor 0xEF53
	S_inode_size        [10]byte // Tamaño del inodo
	S_block_size        [10]byte // Tamaño del bloque
	S_firts_ino         [10]byte // Primer inodo libre
	S_first_blo         [10]byte // Primer bloque libre
	S_bm_inode_start    [10]byte // Guardará el inicio del bitmap de inodos
	S_bm_block_start    [10]byte // Guardará el inicio del bitmap de bloques
	S_inode_start       [10]byte // Guardará el inicio de la tabla de inodos
	S_block_start       [10]byte // Guardará el inicio de la tabla de bloques
}

type Inodos = struct {
	I_uid   [4]byte  // UID del usuario propietario del archivo o carpeta
	I_gid   [4]byte  // GID del grupo al que pertenece el archivo o carpeta.
	I_size  [4]byte  // Tamaño del archivo en bytes
	I_atime [19]byte // Última fecha en que se leyó el inodo sin modificarlo
	I_ctime [19]byte // Fecha en la que se creó el inodo
	I_mtime [19]byte // Última fecha en la que se modifica el inodo
	I_block [4]byte  // Array en los que los primeros 16 registros son bloques directos.
	I_type  [1]byte  // indica si es archivo o carpeta. 1 = Archivo y 0 = Carpeta
	I_perm  [4]byte  // Guardará los permisos del archivo o carpeta.
}

type Content = struct {
	B_name  [10]byte //Nombre de la carpeta o archivo
	B_inodo [10]byte //Apuntador hacia un inodo asociado al archivo o carpeta
}

type BloqueCarpetas = struct {
	B_content [4]Content //Array con el contenido de la carpeta
}

type BloqueArchivos = struct {
	B_content [10]byte // Array con el contenido del archivo
}

type BitMapInodo = struct {
	Status [1]byte
}

type BitmapBloque = struct {
	Status [1]byte
}

// Esto ayuda para el montaje de las particiones

type NodoMount struct {
	id               string
	ruta             string
	nombreparticion  string
	tipoparticion    string
	inicioparticion  int
	tamanioparticion int
	horamontado      string
	numerodisco      int
	nextmount        *NodoMount
	prevmount        *NodoMount
}

var miLista *ListaDobleEnlazada = &ListaDobleEnlazada{}

func main() {
	analizar()
}

func msg_error(err error) {
	fmt.Println("Error: ", err)
}

func analizar() {
	finalizar := false
	fmt.Println("*----------------------------------------------------------*")
	fmt.Println("*                      [MIA] Proyecto 2                    *")
	fmt.Println("*           Cesar Andre Ramirez Davila 202010816           *")
	fmt.Println("*----------------------------------------------------------*")
	reader := bufio.NewReader(os.Stdin)
	//  Ciclo para lectura de multiples comandos
	for !finalizar {
		fmt.Print("Ingrese un comando - ")
		comando, _ := reader.ReadString('\n')
		if strings.Contains(comando, "exit") {
			finalizar = true
		} else {
			if comando != "" && comando != "exit\n" {
				//  Separacion de comando y parametros
				split_comando(comando)
			}
		}
	}
}

func split_comando(comando string) {
	var commandArray []string
	// Eliminacion de saltos de linea
	comando = strings.Replace(comando, "\n", "", 1)
	comando = strings.Replace(comando, "\r", "", 1)

	// Guardado de parametros
	if strings.Contains(comando, "mostrar") {
		commandArray = append(commandArray, comando)
	} else {
		commandArray = strings.Split(comando, " ")
	}
	// Ejecicion de comando leido
	ejecucion_comando(commandArray)
}

func ejecucion_comando(commandArray []string) {
	// Identificacion de comando y ejecucion
	data := strings.ToLower(commandArray[0])
	if data == "mkdisk" {
		comando_mkdisk(commandArray)
	} else if data == "rmdisk" {
		comando_rmdisk(commandArray)
	} else if data == "fdisk" {
		comando_fkdisk(commandArray)
	} else if data == "mount" {
		comando_mount(commandArray)
	} else if data == "mkfs" {
		comando_mkfs(commandArray)
	} else if data == "execute" {
		comando_execute(commandArray)
	} else {
		fmt.Println("Comando ingresado no es valido")
	}
}

func comando_mkdisk(commandArray []string) {
	Disco := MBR{}
	straux := ""
	stamano := ""
	// m_tamano := ""
	m_fecha := ""
	// m_dsk := ""
	m_fit := ""
	tamano := 0
	dimensional := ""
	ajuste := ""
	fit := ""
	ruta := ""
	tamano_archivo := 0
	limite := 0
	bloque := make([]byte, 1024)
	// Lectura de parametros del comando
	for i := 0; i < len(commandArray); i++ {
		//data := strings.ToLower(commandArray[i])
		data := commandArray[i]
		// if strings.HasPrefix(data, ">") {
		// 	// Convertir a minúsculas
		// 	data = strings.ToLower(data)
		// }
		if strings.Contains(data, ">size=") {
			strtam := strings.Replace(data, ">size=", "", 1)
			strtam = strings.Replace(strtam, "\"", "", 2)
			strtam = strings.Replace(strtam, "\r", "", 1)
			stamano = strtam
			tamano2, err := strconv.Atoi(strtam)
			tamano = tamano2
			if err != nil {
				msg_error(err)
			}
		} else if strings.Contains(data, ">unit=") {
			straux = strings.Replace(data, ">unit=", "", 1)
			straux = strings.Replace(dimensional, "\"", "", 2)
			dimensional = straux
		} else if strings.Contains(data, ">fit=") {
			straux = strings.Replace(data, ">fit=", "", 1)
			straux = strings.Replace(ajuste, "\"", "", 2)
			ajuste = straux
		} else if strings.Contains(data, ">path=") {
			ruta = strings.Replace(data, ">path=", "", 1)
			//ruta = data
			//fmt.Println("Ahora? ", ruta)
		}
	}

	//Validando que vengan los parametros obligatorios

	if ruta == "" {
		fmt.Println("¡¡ Error !! No se ha especificado una ruta para crear el disco")
		fmt.Println("")
		return
	}

	if stamano == "" {
		fmt.Println("¡¡ Error !! No se ha especificado el tamanio del disco")
		fmt.Println("")
		return
	}

	rand.Seed(time.Now().UnixNano())

	dsk_s := rand.Intn(1000) + 1

	// Calculo de tamaño del archivo
	if strings.Contains(dimensional, "k") || strings.Contains(dimensional, "K") {
		dimensional = "K"
		tamano_archivo = tamano
	} else if strings.Contains(dimensional, "m") || strings.Contains(dimensional, "M") {
		dimensional = "M"
		tamano_archivo = tamano * 1024
	} else if strings.Contains(dimensional, "") {
		dimensional = "M"
		tamano_archivo = tamano * 1024
	}

	if strings.Contains(ajuste, "ff") {
		fit = "F"
	} else if strings.Contains(ajuste, "bf") {
		fit = "B"
	} else if strings.Contains(ajuste, "wf") {
		fit = "W"
	} else if strings.Contains(ajuste, "") {
		fit = "F"
	}
	// Preparacion del bloque a escribir en archivo
	for j := 0; j < 1024; j++ {
		bloque[j] = 0
	}

	// Creacion, escritura y cierre de archivo
	directorio := path.Dir(ruta)

	if _, err := os.Stat(directorio); os.IsNotExist(err) {
		// la ruta no existe, se debe crear
		if err := os.MkdirAll(directorio, 0755); err != nil {
			panic(err)
		}

		if err := os.Chmod(directorio, 0777); err != nil {
			panic(err)
		}
	} else {
		// la ruta ya existe, se puede continuar
	}

	disco, err := os.Create(ruta)

	if err != nil {
		msg_error(err)
	}

	for limite < tamano_archivo {
		_, err := disco.Write(bloque)
		if err != nil {
			msg_error(err)
		}
		limite++
	}

	if err := disco.Chmod(0777); err != nil {
		panic(err)
	}

	// Obtiene la fecha y hora actual
	fecha_creacion := time.Now().Format("2006-01-02 15:04:05")
	//fmt.Println(fecha_creacion)

	m_fecha = string(fecha_creacion)
	m_fit = string(fit)

	// CONFIGURACION MBR

	copy(Disco.Mbr_tamano[:], strconv.Itoa(tamano_archivo))
	copy(Disco.Mbr_fecha_creacion[:], m_fecha)
	copy(Disco.Mbr_dsk_signature[:], strconv.Itoa(dsk_s))
	copy(Disco.Dsk_fit[:], m_fit)

	//CONFIGURANDO PARTICIONES

	copy(Disco.Mbr_partition_1.Part_status[:], "0")
	copy(Disco.Mbr_partition_2.Part_status[:], "0")
	copy(Disco.Mbr_partition_3.Part_status[:], "0")
	copy(Disco.Mbr_partition_4.Part_status[:], "0")

	copy(Disco.Mbr_partition_1.Part_type[:], "0")
	copy(Disco.Mbr_partition_2.Part_type[:], "0")
	copy(Disco.Mbr_partition_3.Part_type[:], "0")
	copy(Disco.Mbr_partition_4.Part_type[:], "0")

	copy(Disco.Mbr_partition_1.Part_fit[:], "0")
	copy(Disco.Mbr_partition_2.Part_fit[:], "0")
	copy(Disco.Mbr_partition_2.Part_fit[:], "0")
	copy(Disco.Mbr_partition_2.Part_fit[:], "0")

	copy(Disco.Mbr_partition_1.Part_start[:], strconv.Itoa(0))
	copy(Disco.Mbr_partition_2.Part_start[:], strconv.Itoa(0))
	copy(Disco.Mbr_partition_3.Part_start[:], strconv.Itoa(0))
	copy(Disco.Mbr_partition_4.Part_start[:], strconv.Itoa(0))

	copy(Disco.Mbr_partition_1.Part_size[:], strconv.Itoa(0))
	copy(Disco.Mbr_partition_2.Part_size[:], strconv.Itoa(0))
	copy(Disco.Mbr_partition_3.Part_size[:], strconv.Itoa(0))
	copy(Disco.Mbr_partition_4.Part_size[:], strconv.Itoa(0))

	copy(Disco.Mbr_partition_1.Part_name[:], "")
	copy(Disco.Mbr_partition_2.Part_name[:], "")
	copy(Disco.Mbr_partition_3.Part_name[:], "")
	copy(Disco.Mbr_partition_4.Part_name[:], "")

	// // Conversion de struct a bytes
	// ejmbyte := struct_to_bytes(Disco)
	// // Cambio de posicion de puntero dentro del archivo
	// newpos, err := disco.Seek(0, os.SEEK_SET)
	// if err != nil {
	// 	msg_error(err)
	// }
	// // Escritura de struct en archivo binario
	// _, err = disco.WriteAt(ejmbyte, newpos)
	// if err != nil {
	// 	msg_error(err)
	// }

	file, err := os.OpenFile(ruta, os.O_RDWR, 0644)
	if err != nil {
		fmt.Println("¡¡ Error !! No se pudo acceder al disco")
		return
	}
	defer file.Close()

	if _, err := file.Seek(int64(0), 0); err != nil {
		fmt.Println(err)
		return
	}
	if err := binary.Write(file, binary.LittleEndian, &Disco); err != nil {
		fmt.Println(err)
		return
	}

	disco.Close()

	// cout << "" << endl;
	// cout << "*                 Disco creado con exito                   *" << endl;
	// cout << "" << endl;

	fmt.Println("")
	fmt.Println("*                 Disco creado con exito                   *")
	fmt.Println("")

}

func comando_rmdisk(commandArray []string) {
	ruta := ""
	for i := 0; i < len(commandArray); i++ {
		//data := strings.ToLower(commandArray[i])
		data := commandArray[i]
		// if strings.HasPrefix(data, ">") {
		// 	// Convertir a minúsculas
		// 	data = strings.ToLower(data)
		// }
		if strings.Contains(data, ">path=") {
			ruta = strings.Replace(data, ">path=", "", 1)
		}
	}

	if ruta == "" {
		fmt.Println("¡¡ Error !! No se ha especificado una ruta para eliminar")
		fmt.Println("")
		return
	}

	directorio := path.Dir(ruta)
	nombreCompleto := filepath.Base(ruta)
	ultimoDato := strings.Split(nombreCompleto, "/")[len(strings.Split(nombreCompleto, "/"))-1]

	err := filepath.Walk(directorio, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Si el archivo coincide con el nombre que desea eliminar, eliminarlo
		if info.Name() == ultimoDato {
			if err := os.Remove(path); err != nil {
				return err
			}
			fmt.Println("")
			//fmt.Println("*                 Disco creado con exito                   *")
			fmt.Println("*               Disco eliminado con exito                  *")
			fmt.Println("")
		}

		return nil
	})

	if err != nil {
		fmt.Println("¡¡ Error !! No se ha encontrado el archivo", err)
	}

}

/*

*? Faltan las particiones logicas

 */

func comando_fkdisk(commandArray []string) {

	tamano_parti := ""
	straux := ""
	unidad := ""
	rutaa := ""
	tipo_part := ""
	ajuste_part := ""
	nombre_part := ""
	tamano_part := 0
	tamano_archivo1 := 0

	for i := 0; i < len(commandArray); i++ {
		data := commandArray[i]
		// if strings.HasPrefix(data, ">") {
		// 	// Convertir a minúsculas
		// 	data = strings.ToLower(data)
		// }

		if strings.Contains(data, ">size=") {
			strtam := strings.Replace(data, ">size=", "", 1)
			strtam = strings.Replace(strtam, "\"", "", 2)
			strtam = strings.Replace(strtam, "\r", "", 1)
			tamano_parti = strtam
			tamano2, err := strconv.Atoi(strtam)
			tamano_part = tamano2
			if err != nil {
				msg_error(err)
			}
		} else if strings.Contains(data, ">unit=") {
			straux = strings.Replace(data, ">unit=", "", 1)
			//straux = strings.Replace(dimensional, "\"", "", 2)
			unidad = straux
		} else if strings.Contains(data, ">path=") {
			rutaa = strings.Replace(data, ">path=", "", 1)
			//ruta = data
			//fmt.Println("Ahora? ", ruta)
		} else if strings.Contains(data, ">type=") {
			tipo_part = strings.Replace(data, ">type=", "", 1)
		} else if strings.Contains(data, ">fit=") {
			straux = strings.Replace(data, ">fit=", "", 1)
			straux = strings.Replace(data, "\"", "", 2)
			ajuste_part = straux
		} else if strings.Contains(data, ">name=") {
			nombre_part = strings.Replace(data, ">name=", "", 1)
		}
	}

	if tamano_parti == "" {
		fmt.Println("¡¡ Error !! No se ha especificado un tamano para la particion")
		fmt.Println("")
		return
	}

	if rutaa == "" {
		fmt.Println("¡¡ Error !! No se ha especificado una ruta para el disco")
		fmt.Println("")
		return
	}

	if nombre_part == "" {
		fmt.Println("¡¡ Error !! No se ha especificado un nombre para la particion")
		fmt.Println("")
		return
	}

	//part_unit := ""
	part_type := ""
	part_fit := ""

	if strings.Contains(unidad, "b") || strings.Contains(unidad, "B") {
		//part_unit = "B"
		tamano_archivo1 = tamano_part / 1000
	} else if strings.Contains(unidad, "k") || strings.Contains(unidad, "K") {
		//part_unit = "K"
		tamano_archivo1 = tamano_part
	} else if strings.Contains(unidad, "m") || strings.Contains(unidad, "M") {
		//part_unit = "M"
		tamano_archivo1 = tamano_part * 1024
	} else if strings.Contains(unidad, "") {
		//part_unit = "K"
		tamano_archivo1 = tamano_part
	}

	if strings.Contains(tipo_part, "p") || strings.Contains(tipo_part, "P") {
		part_type = "P"
	} else if strings.Contains(tipo_part, "e") || strings.Contains(tipo_part, "E") {
		part_type = "E"
	} else if strings.Contains(tipo_part, "l") || strings.Contains(tipo_part, "L") {
		part_type = "L"
	} else if strings.Contains(tipo_part, "") {
		part_type = "P"
	}

	if strings.Contains(ajuste_part, "bf") {
		part_fit = "B"
	} else if strings.Contains(ajuste_part, "ff") {
		part_fit = "F"
	} else if strings.Contains(ajuste_part, "wf") {
		part_fit = "W"
	} else if strings.Contains(ajuste_part, "") {
		part_fit = "W"
	}

	file, err := os.OpenFile(rutaa, os.O_RDONLY, 0644)
	if err != nil {
		fmt.Println("¡¡ Error !! No se pudo acceder al disco")
		return
	}
	defer file.Close()

	if _, err := file.Seek(int64(0), 0); err != nil {
		fmt.Println(err)
		return
	}

	var PurbeaD MBR
	if err := binary.Read(file, binary.LittleEndian, &PurbeaD); err != nil {
		fmt.Println(err)
		return
	}

	fsType := string(bytes.TrimRight(PurbeaD.Dsk_fit[:], string(0)))
	fsType1 := string(bytes.TrimRight(PurbeaD.Mbr_dsk_signature[:], string(0)))
	fsType2 := string(bytes.TrimRight(PurbeaD.Mbr_fecha_creacion[:], string(0)))
	string_tamano := string(bytes.TrimRight(PurbeaD.Mbr_tamano[:], string(0)))
	fmt.Println("AJuste", fsType)
	fmt.Println("dsk", fsType1)
	fmt.Println("Fecha", fsType2)
	fmt.Println("Tam : ", string_tamano)

	//verDisco(rutaa)

	// fin_archivo := false
	// var emptymbr [4]byte
	// ejm_empty := MBR{}
	// // Apertura de archivo
	// disco, err := os.OpenFile(rutaa, os.O_RDWR, 0660)
	// if err != nil {
	// 	msg_error(err)
	// }
	// // Calculo del tamano de struct en bytes

	// string_tamano := ""
	// var solaa MBR

	// mbr2 := struct_to_bytes(ejm_empty)
	// sstruct := len(mbr2)
	// if !fin_archivo {
	// 	// Lectrura de conjunto de bytes en archivo binario
	// 	lectura := make([]byte, sstruct)
	// 	_, err = disco.ReadAt(lectura, 0)
	// 	if err != nil && err != io.EOF {
	// 		msg_error(err)
	// 	}
	// 	// Conversion de bytes a struct
	// 	mbr := bytes_to_struct_mbr(lectura)

	// 	solaa = mbr
	// 	sstruct = len(lectura)
	// 	if err != nil {
	// 		msg_error(err)
	// 	}
	// 	if mbr.Mbr_tamano == emptymbr {
	// 		fin_archivo = true
	// 	} else {
	// 		string_tamano = string(mbr.Mbr_tamano[:])
	// 	}
	// }
	// disco.Close()

	trimmed_string_tamano := strings.TrimRightFunc(string_tamano, func(r rune) bool { return r == '\x00' })
	tamano, err := strconv.Atoi(trimmed_string_tamano)
	if err != nil {
		fmt.Println("Error:", err)
	}

	//fmt.Println(reflect.TypeOf(tamano))
	//discop := solaa
	discop := PurbeaD

	particion := [4]Partition{
		discop.Mbr_partition_1,
		discop.Mbr_partition_2,
		discop.Mbr_partition_3,
		discop.Mbr_partition_4,
	}

	if tamano >= tamano_archivo1 {

		if part_type != "L" {

			if part_type == "E" {
				for i := 0; i < 4; i++ {
					obtenertipo := ""
					obtenertipo = string(particion[i].Part_type[:])
					obtenertipo = strings.TrimRightFunc(obtenertipo, func(r rune) bool { return r == '\x00' })
					if obtenertipo == "E" {
						fmt.Println("¡¡ Error !! Ya existe una particion extendida")
						return
					}
				}
			}

			nombre_part = strings.TrimRightFunc(nombre_part, func(r rune) bool { return r == '\x00' })
			for i := 0; i < 4; i++ {
				verificrnombre := ""
				verificrnombre = string(particion[i].Part_name[:])
				verificrnombre = strings.TrimRightFunc(verificrnombre, func(r rune) bool { return r == '\x00' })
				if verificrnombre == nombre_part {
					fmt.Println("¡¡ Error !! Ya existe una particion con ese nombre")
					return
				}
			}

			for i := 0; i < 4; i++ {
				//particion[i] = disco.particion[i]
				pruebaa := string(particion[i].Part_name[:])
				otroo := string(particion[i].Part_status[:])
				otroo1 := string(particion[i].Part_fit[:])
				otroo2 := string(particion[i].Part_size[:])
				otroo3 := string(particion[i].Part_start[:])
				otroo4 := string(particion[i].Part_type[:])
				// hacer algo con pruebaa y otroo
				fmt.Println("Nombre ", pruebaa)
				fmt.Println("Status ", otroo)
				fmt.Println("Ajuste ", otroo1)
				fmt.Println("Tamano ", otroo2)
				fmt.Println("Inicio ", otroo3)
				fmt.Println("Tipo ", otroo4)
				fmt.Println("")
			}

			existetipo := ""

			for i := 0; i < 4; i++ {
				existetipo = string(particion[i].Part_type[:])
				existetipo = strings.TrimRightFunc(existetipo, func(r rune) bool { return r == '\x00' })
				if existetipo == "P" || existetipo == "E" {
					switch i {
					case 0:
						discop.Mbr_partition_1.Part_status = [1]byte{'1'}
						//fmt.Println("Entra al 1")
					case 1:
						discop.Mbr_partition_2.Part_status = [1]byte{'1'}
						//fmt.Println("Entra al 2")
					case 2:
						discop.Mbr_partition_3.Part_status = [1]byte{'1'}
						//fmt.Println("Entra al 3")
					case 3:
						discop.Mbr_partition_4.Part_status = [1]byte{'1'}
						//fmt.Println("Entra al 4")
					}
				}
			}

			str_prueba := string(discop.Mbr_partition_1.Part_status[:])
			str_prueba1 := string(discop.Mbr_partition_2.Part_status[:])
			str_prueba2 := string(discop.Mbr_partition_3.Part_status[:])
			str_prueba3 := string(discop.Mbr_partition_4.Part_status[:])
			status_part1 := strings.TrimRightFunc(str_prueba, func(r rune) bool { return r == '\x00' })
			status_part2 := strings.TrimRightFunc(str_prueba1, func(r rune) bool { return r == '\x00' })
			status_part3 := strings.TrimRightFunc(str_prueba2, func(r rune) bool { return r == '\x00' })
			status_part4 := strings.TrimRightFunc(str_prueba3, func(r rune) bool { return r == '\x00' })

			if status_part1 == "0" && status_part2 == "0" && status_part3 == "0" && status_part4 == "0" {
				//fmt.Println("Sigue en 0 el status")
				tamanoDisponibleAntes := tamano
				if tamanoDisponibleAntes >= tamano_archivo1 {
					//fmt.Println("Si hay espacio para la particion")
					copy(discop.Mbr_partition_1.Part_status[:], "0")
					copy(discop.Mbr_partition_1.Part_type[:], part_type)
					copy(discop.Mbr_partition_1.Part_fit[:], part_fit)
					//copy(discop.Mbr_partition_1.Part_start[:], strconv.Itoa(0))
					startt := int(unsafe.Sizeof(discop) + 1)
					copy(discop.Mbr_partition_1.Part_start[:], strconv.Itoa(startt))
					//int(unsafe.Sizeof(discop)+1)
					copy(discop.Mbr_partition_1.Part_size[:], strconv.Itoa(tamano_archivo1))
					copy(discop.Mbr_partition_1.Part_name[:], nombre_part)

					// Apertura del archivo
					// discoescritura, err := os.OpenFile(rutaa, os.O_RDWR, 0660)
					// if err != nil {
					// 	msg_error(err)
					// }

					// // Conversion de struct a bytes
					// ejmbyte := struct_to_bytes(discop)
					// // Cambio de posicion de puntero dentro del archivo
					// newpos, err := discoescritura.Seek(0, os.SEEK_SET)
					// if err != nil {
					// 	msg_error(err)
					// }
					// // Escritura de struct en archivo binario
					// _, err = discoescritura.WriteAt(ejmbyte, newpos)
					// if err != nil {
					// 	msg_error(err)
					// }

					// discoescritura.Close()

					file, err := os.OpenFile(rutaa, os.O_RDWR, 0644)
					if err != nil {
						fmt.Println("¡¡ Error !! No se pudo acceder al disco")
						return
					}
					defer file.Close()

					if _, err := file.Seek(int64(0), 0); err != nil {
						fmt.Println(err)
						return
					}
					if err := binary.Write(file, binary.LittleEndian, &discop); err != nil {
						fmt.Println(err)
						return
					}

					fmt.Println("")
					fmt.Println("*                  Particion 1 asignada                       *")
					fmt.Println("")

				} else {
					fmt.Println("¡¡ Error !! El tamano de la particion es mayor al disponible del disco")
					fmt.Println(tamano)
					fmt.Println(tamano_archivo1)
					return
				}

				// 	tamanoDisponible = disco.mbr_tamano - disco.mbr_partition_1.part_s;
			} else if status_part1 == "1" && status_part2 == "0" && status_part3 == "0" && status_part4 == "0" {
				partSizeStr := string(particion[0].Part_size[:])
				partSizeStr = strings.TrimRightFunc(partSizeStr, func(r rune) bool { return r == '\x00' })
				partSizeInt, err := strconv.Atoi(partSizeStr)
				pruebainicio := (partSizeInt * 1024)
				fmt.Println("Que sale en bytes ", pruebainicio)
				if err != nil {
					// Manejo del error
				}
				tamanoDisponibleAntes1 := tamano - partSizeInt
				// fmt.Println("Tamano disponible? ", tamanoDisponibleAntes1)
				// fmt.Println("Tamano a asignar ", tamano_archivo1)
				// resta := tamanoDisponibleAntes1 - tamano_archivo1
				// fmt.Println("Resta ", resta)
				if tamanoDisponibleAntes1 >= tamano_archivo1 {
					//fmt.Println("Si hay espacio para la particion")
					copy(discop.Mbr_partition_1.Part_status[:], "0")
					copy(discop.Mbr_partition_2.Part_status[:], "0")
					copy(discop.Mbr_partition_2.Part_type[:], part_type)
					copy(discop.Mbr_partition_2.Part_fit[:], part_fit)
					unaprueba := string(particion[0].Part_start[:])
					unaprueba = strings.TrimRightFunc(unaprueba, func(r rune) bool { return r == '\x00' })
					intprueba, err := strconv.Atoi(unaprueba)
					startt := intprueba + pruebainicio + 1
					//startt := intprueba + partSizeInt + 1
					fmt.Println("Inicio? ", intprueba)
					// fmt.Println("Size? ", partSizeInt)
					fmt.Println("Start? ", startt)
					copy(discop.Mbr_partition_2.Part_start[:], strconv.Itoa(startt))
					//disco.mbr_partition_2.part_start = (disco.mbr_partition_1.part_start + disco.mbr_partition_1.part_s + 1);
					copy(discop.Mbr_partition_2.Part_size[:], strconv.Itoa(tamano_archivo1))
					copy(discop.Mbr_partition_2.Part_name[:], nombre_part)

					// //Apertura del archivo
					// discoescritura, err := os.OpenFile(rutaa, os.O_RDWR, 0660)
					// if err != nil {
					// 	msg_error(err)
					// }

					// // Conversion de struct a bytes
					// ejmbyte2 := struct_to_bytes(discop)
					// // Cambio de posicion de puntero dentro del archivo
					// newpos2, err := discoescritura.Seek(0, os.SEEK_SET)
					// if err != nil {
					// 	msg_error(err)
					// }
					// // Escritura de struct en archivo binario
					// _, err = discoescritura.WriteAt(ejmbyte2, newpos2)
					// if err != nil {
					// 	msg_error(err)
					// }

					// discoescritura.Close()

					file, err := os.OpenFile(rutaa, os.O_RDWR, 0644)
					if err != nil {
						fmt.Println("¡¡ Error !! No se pudo acceder al disco")
						return
					}
					defer file.Close()

					if _, err := file.Seek(int64(0), 0); err != nil {
						fmt.Println(err)
						return
					}
					if err := binary.Write(file, binary.LittleEndian, &discop); err != nil {
						fmt.Println(err)
						return
					}

					fmt.Println("")
					fmt.Println("*                  Particion 2 asignada                       *")
					fmt.Println("")
				} else {
					fmt.Println("¡¡ Error !! El tamano de la particion es mayor al disponible del disco")
					fmt.Println(tamano)
					fmt.Println(tamano_archivo1)
					return
				}

			} else if status_part1 == "1" && status_part2 == "1" && status_part3 == "0" && status_part4 == "0" {
				partSizeStr := string(particion[0].Part_size[:])
				partSizeStr1 := string(particion[1].Part_size[:])
				partSizeStr = strings.TrimRightFunc(partSizeStr, func(r rune) bool { return r == '\x00' })
				partSizeStr1 = strings.TrimRightFunc(partSizeStr1, func(r rune) bool { return r == '\x00' })
				partSizeInt, err := strconv.Atoi(partSizeStr)
				partSizeInt1, err := strconv.Atoi(partSizeStr1)
				pruebainicio := (partSizeInt1 * 1024)
				if err != nil {
					// Manejo del error
				}
				tamanoDisponibleAntes1 := tamano - (partSizeInt + partSizeInt1)
				// fmt.Println("Tamano disponible? ", tamanoDisponibleAntes1)
				// fmt.Println("Tamano a asignar ", tamano_archivo1)
				// resta := tamanoDisponibleAntes1 - tamano_archivo1
				// fmt.Println("Resta ", resta)
				if tamanoDisponibleAntes1 >= tamano_archivo1 {
					//fmt.Println("Si hay espacio para la particion")
					copy(discop.Mbr_partition_1.Part_status[:], "0")
					copy(discop.Mbr_partition_2.Part_status[:], "0")
					copy(discop.Mbr_partition_3.Part_status[:], "0")
					copy(discop.Mbr_partition_3.Part_type[:], part_type)
					copy(discop.Mbr_partition_3.Part_fit[:], part_fit)
					unaprueba := string(particion[1].Part_start[:])
					unaprueba = strings.TrimRightFunc(unaprueba, func(r rune) bool { return r == '\x00' })
					intprueba, err := strconv.Atoi(unaprueba)
					startt := intprueba + pruebainicio + 1
					// fmt.Println("Inicio? ", intprueba)
					// fmt.Println("Size? ", partSizeInt)
					// fmt.Println("Start? ", startt)
					copy(discop.Mbr_partition_3.Part_start[:], strconv.Itoa(startt))
					copy(discop.Mbr_partition_3.Part_size[:], strconv.Itoa(tamano_archivo1))
					copy(discop.Mbr_partition_3.Part_name[:], nombre_part)

					// //Apertura del archivo
					// discoescritura, err := os.OpenFile(rutaa, os.O_RDWR, 0660)
					// if err != nil {
					// 	msg_error(err)
					// }

					// // Conversion de struct a bytes
					// ejmbyte3 := struct_to_bytes(discop)
					// // Cambio de posicion de puntero dentro del archivo
					// newpos3, err := discoescritura.Seek(0, os.SEEK_SET)
					// if err != nil {
					// 	msg_error(err)
					// }
					// // Escritura de struct en archivo binario
					// _, err = discoescritura.WriteAt(ejmbyte3, newpos3)
					// if err != nil {
					// 	msg_error(err)
					// }

					//discoescritura.Close()

					file, err := os.OpenFile(rutaa, os.O_RDWR, 0644)
					if err != nil {
						fmt.Println("¡¡ Error !! No se pudo acceder al disco")
						return
					}
					defer file.Close()

					if _, err := file.Seek(int64(0), 0); err != nil {
						fmt.Println(err)
						return
					}
					if err := binary.Write(file, binary.LittleEndian, &discop); err != nil {
						fmt.Println(err)
						return
					}

					fmt.Println("")
					fmt.Println("*                  Particion 3 asignada                       *")
					fmt.Println("")
				} else {
					fmt.Println("¡¡ Error !! El tamano de la particion es mayor al disponible del disco")
					fmt.Println(tamano)
					fmt.Println(tamano_archivo1)
					return
				}
			} else if status_part1 == "1" && status_part2 == "1" && status_part3 == "1" && status_part4 == "0" {
				partSizeStr := string(particion[0].Part_size[:])
				partSizeStr1 := string(particion[1].Part_size[:])
				partSizeStr2 := string(particion[2].Part_size[:])
				partSizeStr = strings.TrimRightFunc(partSizeStr, func(r rune) bool { return r == '\x00' })
				partSizeStr1 = strings.TrimRightFunc(partSizeStr1, func(r rune) bool { return r == '\x00' })
				partSizeStr2 = strings.TrimRightFunc(partSizeStr2, func(r rune) bool { return r == '\x00' })
				partSizeInt, err := strconv.Atoi(partSizeStr)
				partSizeInt1, err := strconv.Atoi(partSizeStr1)
				partSizeInt2, err := strconv.Atoi(partSizeStr2)
				pruebainicio := (partSizeInt2 * 1024)
				if err != nil {
					// Manejo del error
				}
				tamanoDisponibleAntes1 := tamano - (partSizeInt + partSizeInt1 + partSizeInt2)
				// fmt.Println("Tamano disponible? ", tamanoDisponibleAntes1)
				// fmt.Println("Tamano a asignar ", tamano_archivo1)
				// resta := tamanoDisponibleAntes1 - tamano_archivo1
				// fmt.Println("Resta ", resta)
				if tamanoDisponibleAntes1 >= tamano_archivo1 {
					//fmt.Println("Si hay espacio para la particion")
					copy(discop.Mbr_partition_1.Part_status[:], "0")
					copy(discop.Mbr_partition_2.Part_status[:], "0")
					copy(discop.Mbr_partition_3.Part_status[:], "0")
					copy(discop.Mbr_partition_4.Part_status[:], "0")
					copy(discop.Mbr_partition_4.Part_type[:], part_type)
					copy(discop.Mbr_partition_4.Part_fit[:], part_fit)
					unaprueba := string(particion[2].Part_start[:])
					unaprueba = strings.TrimRightFunc(unaprueba, func(r rune) bool { return r == '\x00' })
					intprueba, err := strconv.Atoi(unaprueba)
					startt := intprueba + pruebainicio + 1
					// fmt.Println("Inicio? ", intprueba)
					// fmt.Println("Size? ", partSizeInt)
					// fmt.Println("Start? ", startt)
					copy(discop.Mbr_partition_4.Part_start[:], strconv.Itoa(startt))
					copy(discop.Mbr_partition_4.Part_size[:], strconv.Itoa(tamano_archivo1))
					copy(discop.Mbr_partition_4.Part_name[:], nombre_part)

					// //Apertura del archivo
					// discoescritura, err := os.OpenFile(rutaa, os.O_RDWR, 0660)
					// if err != nil {
					// 	msg_error(err)
					// }

					// // Conversion de struct a bytes
					// ejmbyte4 := struct_to_bytes(discop)
					// // Cambio de posicion de puntero dentro del archivo
					// newpos4, err := discoescritura.Seek(0, os.SEEK_SET)
					// if err != nil {
					// 	msg_error(err)
					// }
					// // Escritura de struct en archivo binario
					// _, err = discoescritura.WriteAt(ejmbyte4, newpos4)
					// if err != nil {
					// 	msg_error(err)
					// }

					// discoescritura.Close()

					file, err := os.OpenFile(rutaa, os.O_RDWR, 0644)
					if err != nil {
						fmt.Println("¡¡ Error !! No se pudo acceder al disco")
						return
					}
					defer file.Close()

					if _, err := file.Seek(int64(0), 0); err != nil {
						fmt.Println(err)
						return
					}
					if err := binary.Write(file, binary.LittleEndian, &discop); err != nil {
						fmt.Println(err)
						return
					}
					fmt.Println("")
					fmt.Println("*                  Particion 4 asignada                       *")
					fmt.Println("")
				} else {
					fmt.Println("¡¡ Error !! El tamano de la particion es mayor al disponible del disco")
					fmt.Println(tamano)
					fmt.Println(tamano_archivo1)
					return
				}
			} else if status_part1 == "1" && status_part2 == "1" && status_part3 == "1" && status_part4 == "1" {
				fmt.Println("¡¡ Error !! Ya no hay particiones disponibles")
				return
			}
		} else if part_type == "L" {
			// fin_archivo := false
			// var emptymbr [10]byte
			// ejm_empty := MBR{}
			// // Apertura de archivo
			// disco, err := os.OpenFile(rutaa, os.O_RDWR, 0660)
			// if err != nil {
			// 	msg_error(err)
			// }
			// // Calculo del tamano de struct en bytes

			// string_tamano := ""
			// //string_fecha := ""
			// //string_dsk := ""
			// //string_ajuste := ""
			// var solaa MBR

			// mbr2 := struct_to_bytes(ejm_empty)
			// sstruct := len(mbr2)
			// if !fin_archivo {
			// 	// Lectrura de conjunto de bytes en archivo binario
			// 	lectura := make([]byte, sstruct)
			// 	_, err = disco.ReadAt(lectura, 0)
			// 	if err != nil && err != io.EOF {
			// 		msg_error(err)
			// 	}
			// 	// Conversion de bytes a struct
			// 	mbr := bytes_to_struct_mbr(lectura)

			// 	solaa = mbr
			// 	sstruct = len(lectura)
			// 	if err != nil {
			// 		msg_error(err)
			// 	}
			// 	if mbr.Mbr_tamano == emptymbr {
			// 		fin_archivo = true
			// 	} else {
			// 		//fmt.Println("Sacando los datos del .dsk")
			// 		//fmt.Print(" Tamaño: ")
			// 		//fmt.Print(string(ejm.Mbr_tamano[:]))
			// 		string_tamano = string(mbr.Mbr_tamano[:])
			// 		//fmt.Print(" Fecha: ")
			// 		//fmt.Print(string(ejm.Mbr_fecha_creacion[:]))
			// 		//string_fecha = string(mbr.Mbr_fecha_creacion[:])
			// 		//fmt.Print(" DSK: ")
			// 		//fmt.Print(string(ejm.Mbr_dsk_signature[:]))
			// 		//string_dsk = string(mbr.Mbr_dsk_signature[:])
			// 		//fmt.Print(" Ajuste: ")
			// 		//fmt.Println(string(ejm.Dsk_fit[:]))
			// 		//string_ajuste = string(mbr.Dsk_fit[:])
			// 	}
			// }
			// disco.Close()

			//Empiezan las particiones logicas
			//Prueba_EBR := EBR{}
			//var obtener_ebr EBR
			encontrado := false
			encontrado1 := false
			encontrado2 := false
			encontrado3 := false
			//for i := 0; i < 4; i++ {
			particion1 := string(particion[0].Part_type[:])
			particion2 := string(particion[1].Part_type[:])
			particion3 := string(particion[2].Part_type[:])
			particion4 := string(particion[3].Part_type[:])
			//strings.TrimRightFunc(obtenertipo, func(r rune) bool { return r == '\x00' })
			particion1 = strings.TrimRightFunc(particion1, func(r rune) bool { return r == '\x00' })
			particion2 = strings.TrimRightFunc(particion2, func(r rune) bool { return r == '\x00' })
			particion3 = strings.TrimRightFunc(particion3, func(r rune) bool { return r == '\x00' })
			particion4 = strings.TrimRightFunc(particion4, func(r rune) bool { return r == '\x00' })

			if particion1 == "E" {
				//fmt.Println("La particion 1 es extendida")
				encontrado = true
			} else if particion2 == "E" {
				//fmt.Println("La particion 2 es extendida")
				encontrado1 = true
			} else if particion3 == "E" {
				//fmt.Println("La particion 3 es extendida")
				encontrado2 = true
			} else if particion4 == "E" {
				//fmt.Println("La particion 4 es extendida")
				encontrado3 = true
			} else {
				fmt.Println("¡¡ Error !! Primero debe crear una particion Extendida")
				return
			}
			//}

			if encontrado == true {

				fmt.Println("Llega ?")
				inicio_particion1 := string(particion[0].Part_start[:])
				inicio_particion1 = strings.TrimRightFunc(inicio_particion1, func(r rune) bool { return r == '\x00' })
				//int_inicio_particion1, err := strconv.Atoi(inicio_particion1)
				int_inicio_particion1, err := strconv.Atoi(inicio_particion1)
				file, err := os.OpenFile(rutaa, os.O_RDONLY, 0644)
				if err != nil {
					fmt.Println("¡¡ Error !! No se pudo acceder al disco")
					return
				}
				defer file.Close()

				if _, err := file.Seek(int64(int_inicio_particion1), 0); err != nil {
					fmt.Println(err)
					return
				}

				var PruebaEBR EBR
				if err := binary.Read(file, binary.LittleEndian, &PruebaEBR); err != nil {
					fmt.Println(err)
					return
				}

				fsType := string(bytes.TrimRight(PruebaEBR.Part_fit[:], string(0)))
				fsType1 := string(bytes.TrimRight(PruebaEBR.Part_name[:], string(0)))
				fsType2 := string(bytes.TrimRight(PruebaEBR.Part_next[:], string(0)))
				string_tamano := string(bytes.TrimRight(PruebaEBR.Part_size[:], string(0)))
				fmt.Println("AJuste", fsType)
				fmt.Println("Nombre", fsType1)
				fmt.Println("Siguiente", fsType2)
				fmt.Println("Tam : ", string_tamano)

				if fsType1 == "" {
					fmt.Println("No hay EBR Escrito")

					inicio_particion1 := string(particion[0].Part_start[:])
					inicio_particion1 = strings.TrimRightFunc(inicio_particion1, func(r rune) bool { return r == '\x00' })
					//int_inicio_particion1, err := strconv.Atoi(inicio_particion1)
					int_inicio_particion1, err := strconv.Atoi(inicio_particion1)

					obtener_ebr := PruebaEBR

					copy(obtener_ebr.Part_status[:], "0")
					copy(obtener_ebr.Part_fit[:], part_fit)
					copy(obtener_ebr.Part_start[:], strconv.Itoa(int_inicio_particion1))
					copy(obtener_ebr.Part_size[:], strconv.Itoa(tamano_archivo1))
					copy(obtener_ebr.Part_next[:], strconv.Itoa(-1))
					copy(obtener_ebr.Part_name[:], nombre_part)

					file, err := os.OpenFile(rutaa, os.O_RDWR, 0644)
					if err != nil {
						fmt.Println("¡¡ Error !! No se pudo acceder al disco")
						return
					}
					defer file.Close()

					if _, err := file.Seek(int64(int_inicio_particion1), 0); err != nil {
						fmt.Println(err)
						return
					}
					if err := binary.Write(file, binary.LittleEndian, &obtener_ebr); err != nil {
						fmt.Println(err)
						return
					}

					fmt.Println("Se ha guardado")
					return
				}
				//var emptyid [100]byte
				// inicio_particion1 := string(particion[0].Part_start[:])
				// inicio_particion1 = strings.TrimRightFunc(inicio_particion1, func(r rune) bool { return r == '\x00' })
				// //int_inicio_particion1, err := strconv.Atoi(inicio_particion1)
				// int_inicio_particion1, err := strconv.Atoi(inicio_particion1)

				// //Veamos
				// // Apertura de archivo
				// disco, err := os.OpenFile(rutaa, os.O_RDWR, 0660)
				// if err != nil {
				// 	msg_error(err)
				// }
				// Calculo del tamano de struct en bytes
				// sstruct := len(struct_to_bytes(ejm_empty))
				// lectura := make([]byte, sstruct)
				// _, err = disco.ReadAt(lectura, int64(int_inicio_particion1))
				// if err != nil && err != io.EOF {
				// 	msg_error(err)
				// }
				// // Conversion de bytes a struct
				// ejm := bytes_to_struct_ebr(lectura)
				// sstruct = len(lectura)
				// if err != nil {
				// 	msg_error(err)
				// } else {
				// 	disco.Close()
				// 	verificarlogicas := string(ejm.Part_name[:])
				// 	verificarlogicas = strings.TrimRightFunc(verificarlogicas, func(r rune) bool { return r == '\x00' })

				// 	if verificarlogicas == "" {
				// 		copy(obtener_ebr.Part_status[:], "0")
				// 		copy(obtener_ebr.Part_fit[:], part_fit)
				// 		copy(obtener_ebr.Part_start[:], strconv.Itoa(int_inicio_particion1))
				// 		copy(obtener_ebr.Part_size[:], strconv.Itoa(tamano_archivo1))
				// 		copy(obtener_ebr.Part_next[:], strconv.Itoa(-1))
				// 		copy(obtener_ebr.Part_name[:], nombre_part)

				// 		fmt.Println("Aun no existe una particion logica")
				// 		fmt.Println("Nombre a poner ", nombre_part)
				// 		fmt.Println("Tamano ", tamano_archivo1)

				// 		// Apertura del archivo
				// 		discoescritura, err := os.OpenFile(rutaa, os.O_RDWR, 0660)
				// 		if err != nil {
				// 			msg_error(err)
				// 		}

				// 		// Conversion de struct a bytes
				// 		ejmbyte := struct_to_bytes(obtener_ebr)
				// 		//prr := len(ejmbyte)
				// 		// Cambio de posicion de puntero dentro del archivo
				// 		newpos, err := discoescritura.Seek(int64(int_inicio_particion1), os.SEEK_SET)
				// 		//startt := int(unsafe.Sizeof(discop) + 1)
				// 		//tamano_ebr := int(unsafe.Sizeof(obtener_ebr))
				// 		//newpos, err := discoescritura.Seek(int64(int_inicio_particion1*tamano_ebr), os.SEEK_SET)
				// 		if err != nil {
				// 			msg_error(err)
				// 		}
				// 		// Escritura de struct en archivo binario
				// 		_, err = discoescritura.WriteAt(ejmbyte, newpos)
				// 		if err != nil {
				// 			msg_error(err)
				// 		}

				// 		discoescritura.Close()

				// 	} else {
				// 		fmt.Println("Datos primer EBR")
				// 		fmt.Println("Nombre ", string(obtener_ebr.Part_name[:]))
				// 		fmt.Println("Tamano ", string(obtener_ebr.Part_size[:]))
				// 		fmt.Println("Inicio ", string(obtener_ebr.Part_start[:]))

				// 		fmt.Println("Entra cuando ya hay alguna particion")
				// 		fmt.Println("Nombre a poner ", nombre_part)
				// 		fmt.Println("Tamano ", tamano_archivo1)
				// 		return
				// 	}
				// }

			} else if encontrado1 == true {
				fmt.Println("Entra al if encontrado 2")
			} else if encontrado2 == true {
				fmt.Println("Entra al if encontrado 3")
			} else if encontrado3 == true {
				fmt.Println("Entra al if encontrado 4")
			}
		}
	} else {
		fmt.Println("¡¡ Error !! El tamano de la particion es mayor al disponible del disco")
		fmt.Println(tamano)
		fmt.Println(tamano_archivo1)
		return
	}
}

func comando_mount(commandArray []string) {
	rutaa := ""
	nombre_part := ""

	for i := 0; i < len(commandArray); i++ {
		data := commandArray[i]
		// if strings.HasPrefix(data, ">") {
		// 	// Convertir a minúsculas
		// 	data = strings.ToLower(data)
		// }

		if strings.Contains(data, ">path=") {
			rutaa = strings.Replace(data, ">path=", "", 1)
		} else if strings.Contains(data, ">name=") {
			nombre_part = strings.Replace(data, ">name=", "", 1)
		}
	}

	if rutaa == "" {
		fmt.Println("¡¡ Error !! No se ha especificado una ruta para el disco")
		fmt.Println("")
		return
	}

	if nombre_part == "" {
		fmt.Println("¡¡ Error !! No se ha especificado un nombre para la particion")
		fmt.Println("")
		return
	}

	file, err := os.OpenFile(rutaa, os.O_RDONLY, 0644)
	if err != nil {
		fmt.Println("¡¡ Error !! No se pudo acceder al disco")
		return
	}
	defer file.Close()

	if _, err := file.Seek(int64(0), 0); err != nil {
		fmt.Println(err)
		return
	}

	var PurbeaD MBR
	if err := binary.Read(file, binary.LittleEndian, &PurbeaD); err != nil {
		fmt.Println(err)
		return
	}

	fsType := string(bytes.TrimRight(PurbeaD.Dsk_fit[:], string(0)))
	fsType1 := string(bytes.TrimRight(PurbeaD.Mbr_dsk_signature[:], string(0)))
	fsType2 := string(bytes.TrimRight(PurbeaD.Mbr_fecha_creacion[:], string(0)))
	string_tamano := string(bytes.TrimRight(PurbeaD.Mbr_tamano[:], string(0)))
	fmt.Println("AJuste", fsType)
	fmt.Println("dsk", fsType1)
	fmt.Println("Fecha", fsType2)
	fmt.Println("Tam : ", string_tamano)

	// fin_archivo := false
	// var emptymbr [4]byte
	// ejm_empty := MBR{}
	// // Apertura de archivo
	// disco, err := os.OpenFile(rutaa, os.O_RDWR, 0660)
	// if err != nil {
	// 	msg_error(err)
	// }
	// // Calculo del tamano de struct en bytes
	// var solaa MBR

	// mbr2 := struct_to_bytes(ejm_empty)
	// sstruct := len(mbr2)
	// if !fin_archivo {
	// 	// Lectrura de conjunto de bytes en archivo binario
	// 	lectura := make([]byte, sstruct)
	// 	_, err = disco.ReadAt(lectura, 0)
	// 	if err != nil && err != io.EOF {
	// 		msg_error(err)
	// 	}
	// 	// Conversion de bytes a struct
	// 	mbr := bytes_to_struct_mbr(lectura)

	// 	solaa = mbr
	// 	sstruct = len(lectura)
	// 	if err != nil {
	// 		msg_error(err)
	// 	}
	// 	if mbr.Mbr_tamano == emptymbr {
	// 		fin_archivo = true
	// 	} else {
	// 		//string_tamano = string(mbr.Mbr_tamano[:])
	// 	}
	// }
	// disco.Close()

	// trimmed_string_tamano := strings.TrimRightFunc(string_tamano, func(r rune) bool { return r == '\x00' })
	// tamano, err := strconv.Atoi(trimmed_string_tamano)
	if err != nil {
		fmt.Println("Error:", err)
	}

	//fmt.Println(reflect.TypeOf(tamano))
	discop := PurbeaD

	particion := [4]Partition{
		discop.Mbr_partition_1,
		discop.Mbr_partition_2,
		discop.Mbr_partition_3,
		discop.Mbr_partition_4,
	}

	name_part1 := string(particion[0].Part_name[:])
	name_part1 = strings.TrimRightFunc(name_part1, func(r rune) bool { return r == '\x00' })
	name_part2 := string(particion[1].Part_name[:])
	name_part2 = strings.TrimRightFunc(name_part2, func(r rune) bool { return r == '\x00' })
	name_part3 := string(particion[2].Part_name[:])
	name_part3 = strings.TrimRightFunc(name_part3, func(r rune) bool { return r == '\x00' })
	name_part4 := string(particion[3].Part_name[:])
	name_part4 = strings.TrimRightFunc(name_part4, func(r rune) bool { return r == '\x00' })

	pruebadefuego := 0
	letraAsignar := ""
	for i := 0; i < 4; i++ {
		saaal := string(particion[i].Part_status[:])
		saaal = strings.TrimRightFunc(saaal, func(r rune) bool { return r == '\x00' })

		if saaal == "1" {
			pruebadefuego++
		}
	}

	if pruebadefuego == 0 {
		letraAsignar = "A"
	} else if pruebadefuego == 1 {
		letraAsignar = "B"
	} else if pruebadefuego == 2 {
		letraAsignar = "C"
	} else if pruebadefuego == 3 {
		letraAsignar = "D"
	}

	ultimosdigitos := "16"
	numeros := obtenerNumeroRutas(miLista)
	//println("Aqui esta la lista ", strconv.Itoa(numeros))
	numeroAsignar := 0

	numerodiscoo := miLista.buscarPorRuta(rutaa)
	//fmt.Println("Que saca esto? ", numerodiscoo)

	if numerodiscoo != 0 {
		numeroAsignar = numerodiscoo
	} else {
		if numeros == 0 {
			numeroAsignar = 1
		} else {
			numeroAsignar = numeros + 1
		}
	}

	//fmt.Println("Cuantos salen? ", strconv.Itoa(pruebadefuego))

	if name_part1 == nombre_part {

		montado := string(discop.Mbr_partition_1.Part_status[:])
		montado = strings.TrimRightFunc(montado, func(r rune) bool { return r == '\x00' })

		if montado == "1" {
			fmt.Println("¡¡ Error !! La particion ya se encuentra montada")
			return
		}

		//fmt.Println("La particion a montar es la 1")

		fecha_creacion := time.Now().Format("2006-01-02 15:04:05")

		fecha_mount := string(fecha_creacion)

		type_part1 := string(particion[0].Part_type[:])
		type_part1 = strings.TrimRightFunc(type_part1, func(r rune) bool { return r == '\x00' })

		start_part1 := string(particion[0].Part_start[:])
		start_part1 = strings.TrimRightFunc(start_part1, func(r rune) bool { return r == '\x00' })
		int_start_part1, err := strconv.Atoi(start_part1)

		size_part1 := string(particion[0].Part_size[:])
		size_part1 = strings.TrimRightFunc(size_part1, func(r rune) bool { return r == '\x00' })
		int_size_part1, err := strconv.Atoi(size_part1)

		if err != nil {
			msg_error(err)
		}
		string_Numero := strconv.Itoa(numeroAsignar)

		nuevonombre := ultimosdigitos + string_Numero + letraAsignar

		miLista.MontarP(nuevonombre, rutaa, nombre_part, type_part1, int_start_part1, int_size_part1, fecha_mount, numeroAsignar)

		copy(discop.Mbr_partition_1.Part_status[:], "1")

		// //Apertura del archivo
		// discoescritura, err := os.OpenFile(rutaa, os.O_RDWR, 0660)
		// if err != nil {
		// 	msg_error(err)
		// }

		// // Conversion de struct a bytes
		// ejmbyte4 := struct_to_bytes(discop)
		// // Cambio de posicion de puntero dentro del archivo
		// newpos4, err := discoescritura.Seek(0, os.SEEK_SET)
		// if err != nil {
		// 	msg_error(err)
		// }
		// // Escritura de struct en archivo binario
		// _, err = discoescritura.WriteAt(ejmbyte4, newpos4)
		// if err != nil {
		// 	msg_error(err)
		// }

		// discoescritura.Close()

		file, err := os.OpenFile(rutaa, os.O_RDWR, 0644)
		if err != nil {
			fmt.Println("¡¡ Error !! No se pudo acceder al disco")
			return
		}
		defer file.Close()

		if _, err := file.Seek(int64(0), 0); err != nil {
			fmt.Println(err)
			return
		}
		if err := binary.Write(file, binary.LittleEndian, &discop); err != nil {
			fmt.Println(err)
			return
		}

		fmt.Println("")
		fmt.Println("*               Particion montada con exito                *")
		fmt.Println("")

		miLista.ImprimirTabla()
	} else if name_part2 == nombre_part {
		montado := string(discop.Mbr_partition_2.Part_status[:])
		montado = strings.TrimRightFunc(montado, func(r rune) bool { return r == '\x00' })

		if montado == "1" {
			fmt.Println("¡¡ Error !! La particion ya se encuentra montada")
			return
		}
		//fmt.Println("La particion a montar es la 2")

		fecha_creacion := time.Now().Format("2006-01-02 15:04:05")

		fecha_mount := string(fecha_creacion)

		type_part1 := string(particion[1].Part_type[:])
		type_part1 = strings.TrimRightFunc(type_part1, func(r rune) bool { return r == '\x00' })

		start_part1 := string(particion[1].Part_start[:])
		start_part1 = strings.TrimRightFunc(start_part1, func(r rune) bool { return r == '\x00' })
		int_start_part1, err := strconv.Atoi(start_part1)

		size_part1 := string(particion[1].Part_size[:])
		size_part1 = strings.TrimRightFunc(size_part1, func(r rune) bool { return r == '\x00' })
		int_size_part1, err := strconv.Atoi(size_part1)

		if err != nil {
			msg_error(err)
		}
		string_Numero := strconv.Itoa(numeroAsignar)

		nuevonombre := ultimosdigitos + string_Numero + letraAsignar

		miLista.MontarP(nuevonombre, rutaa, nombre_part, type_part1, int_start_part1, int_size_part1, fecha_mount, numeroAsignar)

		copy(discop.Mbr_partition_2.Part_status[:], "1")

		// //Apertura del archivo
		// discoescritura, err := os.OpenFile(rutaa, os.O_RDWR, 0660)
		// if err != nil {
		// 	msg_error(err)
		// }

		// // Conversion de struct a bytes
		// ejmbyte4 := struct_to_bytes(discop)
		// // Cambio de posicion de puntero dentro del archivo
		// newpos4, err := discoescritura.Seek(0, os.SEEK_SET)
		// if err != nil {
		// 	msg_error(err)
		// }
		// // Escritura de struct en archivo binario
		// _, err = discoescritura.WriteAt(ejmbyte4, newpos4)
		// if err != nil {
		// 	msg_error(err)
		// }

		// discoescritura.Close()

		file, err := os.OpenFile(rutaa, os.O_RDWR, 0644)
		if err != nil {
			fmt.Println("¡¡ Error !! No se pudo acceder al disco")
			return
		}
		defer file.Close()

		if _, err := file.Seek(int64(0), 0); err != nil {
			fmt.Println(err)
			return
		}
		if err := binary.Write(file, binary.LittleEndian, &discop); err != nil {
			fmt.Println(err)
			return
		}

		fmt.Println("")
		fmt.Println("*               Particion montada con exito                *")
		fmt.Println("")

		miLista.ImprimirTabla()
	} else if name_part3 == nombre_part {
		montado := string(discop.Mbr_partition_3.Part_status[:])
		montado = strings.TrimRightFunc(montado, func(r rune) bool { return r == '\x00' })

		if montado == "1" {
			fmt.Println("¡¡ Error !! La particion ya se encuentra montada")
			return
		}
		//fmt.Println("La particion a montar es la 3")

		fecha_creacion := time.Now().Format("2006-01-02 15:04:05")

		fecha_mount := string(fecha_creacion)

		type_part1 := string(particion[2].Part_type[:])
		type_part1 = strings.TrimRightFunc(type_part1, func(r rune) bool { return r == '\x00' })

		start_part1 := string(particion[2].Part_start[:])
		start_part1 = strings.TrimRightFunc(start_part1, func(r rune) bool { return r == '\x00' })
		int_start_part1, err := strconv.Atoi(start_part1)

		size_part1 := string(particion[2].Part_size[:])
		size_part1 = strings.TrimRightFunc(size_part1, func(r rune) bool { return r == '\x00' })
		int_size_part1, err := strconv.Atoi(size_part1)

		if err != nil {
			msg_error(err)
		}
		string_Numero := strconv.Itoa(numeroAsignar)

		nuevonombre := ultimosdigitos + string_Numero + letraAsignar

		miLista.MontarP(nuevonombre, rutaa, nombre_part, type_part1, int_start_part1, int_size_part1, fecha_mount, numeroAsignar)

		copy(discop.Mbr_partition_3.Part_status[:], "1")

		// //Apertura del archivo
		// discoescritura, err := os.OpenFile(rutaa, os.O_RDWR, 0660)
		// if err != nil {
		// 	msg_error(err)
		// }

		// // Conversion de struct a bytes
		// ejmbyte4 := struct_to_bytes(discop)
		// // Cambio de posicion de puntero dentro del archivo
		// newpos4, err := discoescritura.Seek(0, os.SEEK_SET)
		// if err != nil {
		// 	msg_error(err)
		// }
		// // Escritura de struct en archivo binario
		// _, err = discoescritura.WriteAt(ejmbyte4, newpos4)
		// if err != nil {
		// 	msg_error(err)
		// }

		// discoescritura.Close()

		file, err := os.OpenFile(rutaa, os.O_RDWR, 0644)
		if err != nil {
			fmt.Println("¡¡ Error !! No se pudo acceder al disco")
			return
		}
		defer file.Close()

		if _, err := file.Seek(int64(0), 0); err != nil {
			fmt.Println(err)
			return
		}
		if err := binary.Write(file, binary.LittleEndian, &discop); err != nil {
			fmt.Println(err)
			return
		}

		fmt.Println("")
		fmt.Println("*               Particion montada con exito                *")
		fmt.Println("")

		miLista.ImprimirTabla()
	} else if name_part4 == nombre_part {
		montado := string(discop.Mbr_partition_4.Part_status[:])
		montado = strings.TrimRightFunc(montado, func(r rune) bool { return r == '\x00' })

		if montado == "1" {
			fmt.Println("¡¡ Error !! La particion ya se encuentra montada")
			return
		}
		//fmt.Println("La particion a montar es la 4")

		fecha_creacion := time.Now().Format("2006-01-02 15:04:05")

		fecha_mount := string(fecha_creacion)

		type_part1 := string(particion[3].Part_type[:])
		type_part1 = strings.TrimRightFunc(type_part1, func(r rune) bool { return r == '\x00' })

		start_part1 := string(particion[3].Part_start[:])
		start_part1 = strings.TrimRightFunc(start_part1, func(r rune) bool { return r == '\x00' })
		int_start_part1, err := strconv.Atoi(start_part1)

		size_part1 := string(particion[3].Part_size[:])
		size_part1 = strings.TrimRightFunc(size_part1, func(r rune) bool { return r == '\x00' })
		int_size_part1, err := strconv.Atoi(size_part1)

		if err != nil {
			msg_error(err)
		}
		string_Numero := strconv.Itoa(numeroAsignar)

		nuevonombre := ultimosdigitos + string_Numero + letraAsignar

		miLista.MontarP(nuevonombre, rutaa, nombre_part, type_part1, int_start_part1, int_size_part1, fecha_mount, numeroAsignar)

		copy(discop.Mbr_partition_4.Part_status[:], "1")

		// //Apertura del archivo
		// discoescritura, err := os.OpenFile(rutaa, os.O_RDWR, 0660)
		// if err != nil {
		// 	msg_error(err)
		// }

		// // Conversion de struct a bytes
		// ejmbyte4 := struct_to_bytes(discop)
		// // Cambio de posicion de puntero dentro del archivo
		// newpos4, err := discoescritura.Seek(0, os.SEEK_SET)
		// if err != nil {
		// 	msg_error(err)
		// }
		// // Escritura de struct en archivo binario
		// _, err = discoescritura.WriteAt(ejmbyte4, newpos4)
		// if err != nil {
		// 	msg_error(err)
		// }

		// discoescritura.Close()

		file, err := os.OpenFile(rutaa, os.O_RDWR, 0644)
		if err != nil {
			fmt.Println("¡¡ Error !! No se pudo acceder al disco")
			return
		}
		defer file.Close()

		if _, err := file.Seek(int64(0), 0); err != nil {
			fmt.Println(err)
			return
		}
		if err := binary.Write(file, binary.LittleEndian, &discop); err != nil {
			fmt.Println(err)
			return
		}

		fmt.Println("")
		fmt.Println("*               Particion montada con exito                *")
		fmt.Println("")

		miLista.ImprimirTabla()
	} else {
		fmt.Println("¡¡ Error !! No se encontro una particion con ese nombre")
		return
	}

}

func comando_mkfs(commandArray []string) {
	straux := ""

	id_buscar := ""
	type_part := ""

	for i := 0; i < len(commandArray); i++ {
		data := commandArray[i]
		// if strings.HasPrefix(data, ">") {
		// 	// Convertir a minúsculas
		// 	data = strings.ToLower(data)
		// }

		if strings.Contains(data, ">id=") {
			straux = strings.Replace(data, ">id=", "", 1)
			//straux = strings.Replace(dimensional, "\"", "", 2)
			id_buscar = straux
		} else if strings.Contains(data, ">type=") {
			type_part = strings.Replace(data, ">type=", "", 1)
		}
	}

	if id_buscar == "" {
		fmt.Println("¡¡ Error !! No se ha especificado un id para el formateo")
		fmt.Println("")
		return
	}

	tipo_formateo := ""

	if type_part == "full" {
		tipo_formateo = "full"
	} else if type_part == "" {
		tipo_formateo = "full"
	} else {
		fmt.Println("El tipo de formateo no es aceptado")
		return
	}

	if tipo_formateo == "full" {
		existe, nodo := miLista.buscarPorID(id_buscar)
		if existe {
			//Empieza lo chido del EXT2 DX

			partb := nodo.tamanioparticion * 1024 // tamaño del bloque en bytes
			sb := SuperBloque{}                   // instancia de SuperBloque
			pruebaI := Inodos{}
			size := unsafe.Sizeof(sb)
			size_Inode := unsafe.Sizeof(pruebaI)

			n := float64(partb-int(size)) / (4 + float64(size_Inode) + 3*64)
			nInodos := int(math.Floor(n))

			n_e := float64(nInodos)

			fmt.Println("Que sale en size superbloque ? ", strconv.Itoa(int(size)))
			fmt.Println("Que sale en size inodos ? ", strconv.Itoa(int(size_Inode)))

			fmt.Println(nInodos, n_e)

			crear_EXT2(nodo, int(n_e))

			fmt.Println("")
			fmt.Println("*               Formato EXT2 creado con exito              *")
			fmt.Println("")

		} else {
			fmt.Println("No se encontró ningúna particion con ese ID")
			return
		}
	}
}

func crear_EXT2(nodoActual *NodoMount, n int) {

	// Se crea el SuperBloque
	SP := SuperBloque{}
	pruebaI := Inodos{}
	BloqueC := BloqueCarpetas{}
	size := unsafe.Sizeof(SP)
	size_Inode := unsafe.Sizeof(pruebaI)
	size_BloqueC := unsafe.Sizeof(BloqueC)
	horamontado := nodoActual.horamontado
	magic := "0XEF53"
	first_in := nodoActual.inicioparticion + int(size) + 3*n + n
	first_blo := int(binary.LittleEndian.Uint32(SP.S_firts_ino[:])) + n*int(size_Inode)
	bm_inode_start := nodoActual.inicioparticion + int(size)
	bm_block_start := int(binary.LittleEndian.Uint32(SP.S_bm_inode_start[:])) + n
	inode_star := int(binary.LittleEndian.Uint32(SP.S_firts_ino[:]))
	block_start := int(binary.LittleEndian.Uint32(SP.S_inode_start[:])) + n*int(size_Inode)
	copy(SP.S_filesystem_type[:], strconv.Itoa(2))
	copy(SP.S_inodes_count[:], strconv.Itoa(n))
	copy(SP.S_blocks_count[:], strconv.Itoa(3*n))
	copy(SP.S_free_blocks_count[:], strconv.Itoa(3*n-2))
	copy(SP.S_free_inodes_count[:], strconv.Itoa(n-2))
	copy(SP.S_mtime[:], horamontado)
	copy(SP.S_mnt_count[:], strconv.Itoa(1))
	copy(SP.S_magic[:], magic)
	copy(SP.S_inode_size[:], strconv.Itoa(int(size_Inode)))
	copy(SP.S_block_size[:], strconv.Itoa(int(size_BloqueC)))
	copy(SP.S_firts_ino[:], strconv.Itoa(first_in))
	copy(SP.S_first_blo[:], strconv.Itoa(first_blo))
	copy(SP.S_bm_inode_start[:], strconv.Itoa(bm_inode_start))
	copy(SP.S_bm_block_start[:], strconv.Itoa(bm_block_start))
	copy(SP.S_inode_start[:], strconv.Itoa(inode_star))
	copy(SP.S_block_start[:], strconv.Itoa(block_start))

	Escribir_SuperBloque(nodoActual.ruta, SP, nodoActual.inicioparticion)

	//Se crea el Bitmap de Inodos

	bmInodo := make([]BitMapInodo, n)
	siguientes := BitMapInodo{Status: [1]byte{'0'}}

	bmInodo[0].Status = [1]byte{'1'}
	bmInodo[1].Status = [1]byte{'1'}

	for i := 2; i < n; i++ {
		bmInodo[i] = siguientes
	}

	verSB(nodoActual.ruta, nodoActual.inicioparticion)
	EscribirBitMapInodos(nodoActual.ruta, bmInodo, int64(nodoActual.inicioparticion), n)

	// counter := 0
	// for i := 0; i < n; i++ {
	// 	fmt.Print(string(bmInodo[i].Status[:]), " ")
	// 	counter++
	// 	if counter == 20 {
	// 		fmt.Println()
	// 		counter = 0
	// 	}
	// }
	// if counter != 0 {
	// 	fmt.Println()
	// }

	//Se crea el Bitmap de Bloques

	//fmt.Println("Se crean los de Bloque")

	bmBloque := make([]BitmapBloque, n*3)
	siguientes_bloque := BitmapBloque{Status: [1]byte{'0'}}

	bmBloque[0].Status = [1]byte{'1'}
	bmBloque[1].Status = [1]byte{'1'}

	for i := 2; i < n; i++ {
		bmBloque[i] = siguientes_bloque
	}

	EscribirBitMapBloques(nodoActual.ruta, bmBloque, int64(nodoActual.inicioparticion), n*3, n)

	// counter1 := 0
	// for i := 0; i < n; i++ {
	// 	fmt.Print(string(bmBloque[i].Status[:]), " ")
	// 	counter1++
	// 	if counter1 == 20 {
	// 		fmt.Println()
	// 		counter1 = 0
	// 	}
	// }
	// if counter1 != 0 {
	// 	fmt.Println()
	// }

	// Se crean manualmente los primeros Inodos

	// Predeterminado[0].i_block[0] = 0;
	// Predeterminado[0].i_type = '0';
	// Predeterminado[0].i_perm = 664;

	// I_block [10]byte // Array en los que los primeros 16 registros son bloques directos.
	// I_type  [10]byte // indica si es archivo o carpeta. 1 = Archivo y 0 = Carpeta
	// I_perm  [10]byte // Guardará los permisos del archivo o carpeta.

	fecha_creacion := time.Now().Format("2006-01-02 15:04:05")
	fecha_atime := string(fecha_creacion)
	//avr := ""

	// var avr string = "abcdefghij"
	var fechaBytes [19]byte

	// // Convertir la cadena en un array de bytes de longitud 10
	copy(fechaBytes[:], []byte(fecha_atime))

	Predeterminado := make([]Inodos, 2)
	Predeterminado[0].I_uid = [4]byte{'1'}
	Predeterminado[0].I_gid = [4]byte{'1'}
	Predeterminado[0].I_atime = fechaBytes
	Predeterminado[0].I_ctime = fechaBytes
	Predeterminado[0].I_mtime = fechaBytes
	Predeterminado[0].I_block = [4]byte{'0'}
	Predeterminado[0].I_type = [1]byte{'0'}
	Predeterminado[0].I_perm = [4]byte{'0'}

	uidBytes := Predeterminado[0].I_uid[:]
	gidBytes := Predeterminado[0].I_gid[:]
	atimeBytes := Predeterminado[0].I_atime[:]
	ctimeBytes := Predeterminado[0].I_ctime[:]
	mtimeBytes := Predeterminado[0].I_mtime[:]
	blockBytes := Predeterminado[0].I_block[:]
	typeBytes := Predeterminado[0].I_type[:]
	permBytes := Predeterminado[0].I_perm[:]

	// Copiar los bytes en el slice de destino utilizando el método copy()
	copy(uidBytes, []byte("1"))
	copy(gidBytes, []byte("1"))
	copy(atimeBytes, []byte(fecha_atime))
	copy(ctimeBytes, []byte(fecha_atime))
	copy(mtimeBytes, []byte(fecha_atime))
	copy(blockBytes, []byte("0"))
	copy(typeBytes, []byte("0"))
	copy(permBytes, []byte("0"))

}

func Escribir_SuperBloque(path string, SB SuperBloque, inicioP int) {

	Tam_EBR := EBR{}
	EBR_Size := unsafe.Sizeof(Tam_EBR)

	nuevoInicio := inicioP + int(EBR_Size)

	file, err := os.OpenFile(path, os.O_RDWR, 0644)
	if err != nil {
		fmt.Println("¡¡ Error !! No se pudo acceder al disco")
		return
	}
	defer file.Close()

	if _, err := file.Seek(int64(nuevoInicio), 0); err != nil {
		fmt.Println(err)
		return
	}
	if err := binary.Write(file, binary.LittleEndian, &SB); err != nil {
		fmt.Println(err)
		return
	}

}

func verDisco(path string) {
	file, err := os.OpenFile(path, os.O_RDONLY, 0644)
	if err != nil {
		fmt.Println("¡¡ Error !! No se pudo acceder al disco")
		return
	}
	defer file.Close()

	if _, err := file.Seek(int64(0), 0); err != nil {
		fmt.Println(err)
		return
	}

	var PurbeaD MBR
	if err := binary.Read(file, binary.LittleEndian, &PurbeaD); err != nil {
		fmt.Println(err)
		return
	}

	fsType := string(bytes.TrimRight(PurbeaD.Dsk_fit[:], string(0)))
	fsType1 := string(bytes.TrimRight(PurbeaD.Mbr_dsk_signature[:], string(0)))
	fsType2 := string(bytes.TrimRight(PurbeaD.Mbr_fecha_creacion[:], string(0)))
	fsType3 := string(bytes.TrimRight(PurbeaD.Mbr_tamano[:], string(0)))
	fmt.Println("AJuste", fsType)
	fmt.Println("dsk", fsType1)
	fmt.Println("Fecha", fsType2)
	fmt.Println("Tam : ", fsType3)

	return
}

func verSB(path string, inicioP int) {

	file, err := os.OpenFile(path, os.O_RDONLY, 0644)
	if err != nil {
		fmt.Println("¡¡ Error !! No se pudo acceder al disco")
		return
	}
	defer file.Close()

	if _, err := file.Seek(int64(inicioP), 0); err != nil {
		fmt.Println(err)
		return
	}

	var SB SuperBloque
	if err := binary.Read(file, binary.LittleEndian, &SB); err != nil {
		fmt.Println(err)
		return
	}

	fsType := string(bytes.TrimRight(SB.S_filesystem_type[:], string(0)))
	fsType1 := string(bytes.TrimRight(SB.S_mtime[:], string(0)))
	fsType2 := string(bytes.TrimRight(SB.S_magic[:], string(0)))
	fmt.Println(fsType)
	fmt.Println(fsType1)
	fmt.Println(fsType2)
}

func EscribirBitMapInodos(path string, bmInodo []BitMapInodo, inicio int64, n int) error {

	SB := SuperBloque{}
	prueba := unsafe.Sizeof(SB)

	file, err := os.OpenFile(path, os.O_RDWR, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	for i := 0; i < n; i++ {
		offset := int64(inicio) + int64(i)*int64(binary.Size(BitMapInodo{}))
		offset1 := offset + int64(prueba)
		if _, err := file.Seek(offset1, 0); err != nil {
			return err
		}
		if err := binary.Write(file, binary.LittleEndian, &bmInodo[i]); err != nil {
			return err
		}
	}

	return nil
}

func EscribirBitMapBloques(path string, bmBloque []BitmapBloque, inicio int64, n int, npos int) error {

	SB := SuperBloque{}
	BMInode := BitMapInodo{}

	BmInode_size := unsafe.Sizeof(BMInode)
	prueba := unsafe.Sizeof(SB)

	file, err := os.OpenFile(path, os.O_RDWR, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	for i := 0; i < n; i++ {
		//offset := int64(inicio) + int64(i)*int64(binary.Size(BitMapInodo{}))
		offset := int64(inicio) + int64(i)
		offset1 := offset + int64(prueba) + int64(BmInode_size+uintptr(npos))
		if _, err := file.Seek(offset1, 0); err != nil {
			return err
		}
		if err := binary.Write(file, binary.LittleEndian, &bmBloque[i]); err != nil {
			return err
		}
	}

	return nil
}

func comando_execute(commandArray []string) {
	rutaa := ""

	for i := 0; i < len(commandArray); i++ {
		data := commandArray[i]
		// if strings.HasPrefix(data, ">") {
		// 	// Convertir a minúsculas
		// 	data = strings.ToLower(data)
		// }

		if strings.Contains(data, ">path=") {
			rutaa = strings.Replace(data, ">path=", "", 1)
		}
	}

	if rutaa == "" {
		fmt.Println("¡¡ Error !! No se ha especificado una ruta para el disco")
		fmt.Println("")
		return
	}

	ext := filepath.Ext(rutaa)
	if ext == ".eea" {
		archivo, err := os.Open(rutaa)
		if err != nil {
			fmt.Println(err)
			return
		}
		defer archivo.Close()

		sc := bufio.NewScanner(archivo)
		for sc.Scan() {
			linea := sc.Text()
			if len(linea) > 0 {
				// buscar el índice del símbolo #
				indiceComentario := strings.Index(linea, "#")

				// si hay un comentario, obtenerlo
				if indiceComentario >= 0 {
					comentario := linea[indiceComentario+1:]
					linea = linea[:indiceComentario]

					// imprimir el comentario
					fmt.Println("Comentario encontrado ->", comentario)
				}

				// separar el comando y los argumentos
				comando := strings.TrimSpace(linea)
				if comando != "" && comando != "exit" {
					fmt.Println("*----------------------------------------------------------*")
					fmt.Println("*                      [MIA] Proyecto 2                    *")
					fmt.Println("*           Cesar Andre Ramirez Davila 202010816           *")
					fmt.Println("*----------------------------------------------------------*")
					fmt.Println("Ejecutando el comando -", comando)
					split_comando(comando)
				}
			}
		}

		if err := sc.Err(); err != nil {
			fmt.Println(err)
		}

	} else {
		fmt.Println("¡¡ Error !! La extension del archivo no corresponde a .eea")
	}

}

func struct_to_bytes(p interface{}) []byte {
	// Codificacion de Struct a []Bytes
	buf := bytes.Buffer{}
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(p)
	if err != nil && err != io.EOF {
		msg_error(err)
	}
	return buf.Bytes()
}

func bytes_to_struct_mbr(s []byte) MBR {
	// Decodificacion de [] Bytes a Struct ejemplo
	p := MBR{}
	dec := gob.NewDecoder(bytes.NewReader(s))
	err := dec.Decode(&p)
	if err != nil && err != io.EOF {
		msg_error(err)
	}
	return p
}

func bytes_to_struct_ebr(s []byte) EBR {
	// Decodificacion de [] Bytes a Struct ejemplo
	p := EBR{}
	dec := gob.NewDecoder(bytes.NewReader(s))
	err := dec.Decode(&p)
	if err != nil && err != io.EOF {
		msg_error(err)
	}
	return p
}

func bytes_to_struct_SB(s []byte) SuperBloque {
	// Decodificacion de [] Bytes a Struct ejemplo
	p := SuperBloque{}
	dec := gob.NewDecoder(bytes.NewReader(s))
	err := dec.Decode(&p)
	if err != nil && err != io.EOF {
		msg_error(err)
	}
	return p
}

type ListaDobleEnlazada struct {
	first *NodoMount
	last  *NodoMount
}

func (lista *ListaDobleEnlazada) MontarP(id string, ruta string, nombreparticion string, tipoparticion string, inicioparticion int, tamanioparticion int, horamontado string, numerodisco int) (*NodoMount, error) {

	nuevoNodo := &NodoMount{
		id:               id,
		ruta:             ruta,
		nombreparticion:  nombreparticion,
		tipoparticion:    tipoparticion,
		inicioparticion:  inicioparticion,
		tamanioparticion: tamanioparticion,
		horamontado:      horamontado,
		numerodisco:      numerodisco,
		nextmount:        nil,
		prevmount:        nil,
	}

	if lista.first == nil {
		lista.first = nuevoNodo
		lista.last = nuevoNodo
	} else {
		lista.last.nextmount = nuevoNodo
		nuevoNodo.prevmount = lista.last
		lista.last = nuevoNodo
	}

	return nuevoNodo, nil
}

func (lista *ListaDobleEnlazada) Imprimir() {
	actual := lista.first
	for actual != nil {
		fmt.Printf("id: %s, ruta: %s, nombreparticion: %s, tipoparticion: %s, inicioparticion: %d, tamanioparticion: %d, horamontado: %s\n", actual.id, actual.ruta, actual.nombreparticion, actual.tipoparticion, actual.inicioparticion, actual.tamanioparticion, actual.horamontado)
		actual = actual.nextmount
	}
}

func (lista *ListaDobleEnlazada) ImprimirTabla() {
	// Imprimir encabezado de la tabla
	fmt.Println("Particiones Montadas actualmente")
	fmt.Println("")
	fmt.Printf("%-10s %-20s\n", "ID", "Hora Montado")
	fmt.Println(strings.Repeat("-", 32))

	// Recorrer todos los nodos de la lista
	current := lista.first
	for current != nil {
		// Imprimir los datos correspondientes del nodo
		fmt.Printf("%-10s %-20s\n", current.id, current.horamontado)

		// Avanzar al siguiente nodo
		current = current.nextmount
	}
}

func (lista *ListaDobleEnlazada) EstaVacia() bool {
	return lista.first == nil
}

func contarRutas(lista *ListaDobleEnlazada) int {
	rutasDistintas := make(map[string]bool)

	for nodo := lista.first; nodo != nil; nodo = nodo.nextmount {
		ruta := nodo.ruta
		rutasDistintas[ruta] = true
	}

	return len(rutasDistintas)
}

func obtenerNumeroRutas(lista *ListaDobleEnlazada) int {
	contador := contarRutas(lista)
	return contador
}

func (lista *ListaDobleEnlazada) buscarPorRuta(ruta string) int {
	for nodo := lista.first; nodo != nil; nodo = nodo.nextmount {
		if nodo.ruta == ruta {
			return nodo.numerodisco
		}
	}
	return 0
}

func (lista *ListaDobleEnlazada) buscarPorID(id string) (bool, *NodoMount) {
	for nodo := lista.first; nodo != nil; nodo = nodo.nextmount {
		if nodo.id == id {
			return true, nodo
		}
	}
	return false, nil
}
