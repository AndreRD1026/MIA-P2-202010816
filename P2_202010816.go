package main

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unsafe"

	"github.com/rs/cors"
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
	B_content [64]byte // Array con el contenido del archivo
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

// Esto ayuda a llevar el login

type NodoLogin struct {
	id     string
	nombre string
}

var miLista *ListaDobleEnlazada = &ListaDobleEnlazada{}
var usuarioLogeado []NodoLogin

type cmdstruct struct {
	Cmd string `json:"cmd"`
}

type LoginRequest struct {
	Id   string `json:"id"`
	User string `json:"user"`
	Pass string `json:"pass"`
}

func main() {
	//analizar()

	mux := http.NewServeMux()

	mux.HandleFunc("/analizar", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		var Content cmdstruct
		respuesta := ""
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &Content)
		respuesta = split_comando(Content.Cmd)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"result": "` + respuesta + `" }`))
	})

	mux.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		var Prrueba LoginRequest
		respuesta := ""
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &Prrueba)
		respuesta = nuevologin(Prrueba.Id, Prrueba.User, Prrueba.Pass)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"result_log": "` + respuesta + `" }`))
	})

	mux.HandleFunc("/logout", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		respuesta := ""
		respuesta = comando_logout()
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"result": "` + respuesta + `" }`))
	})

	fmt.Println("Server ON in port 5000")
	handler := cors.Default().Handler(mux)
	log.Fatal(http.ListenAndServe(":5000", handler))
}

func toBase64(b []byte) string {
	return base64.StdEncoding.EncodeToString(b)
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

//Empiezan los cambios
func split_comando(comando string) string {
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
	return ejecucion_comando(commandArray)
}

// func split_comando(comando string) string {
// 	var commandArray []string
// 	// Eliminacion de saltos de linea
// 	comando = strings.Replace(comando, "\n", "", 1)
// 	comando = strings.Replace(comando, "\r", "", 1)

// 	// Guardado de parametros

// 	if strings.Contains(comando, "mostrar") {
// 		// indiceComentario := strings.Index(comando, "#")

// 		// // si hay un comentario, obtenerlo
// 		// if indiceComentario >= 0 {
// 		// 	comentario := comando[indiceComentario+1:]
// 		// 	// imprimir el comentario
// 		// 	fmt.Println("Comentario encontrado ->", comentario)
// 		// }
// 		commandArray = append(commandArray, comando)
// 	} else {
// 		commandArray = strings.Split(comando, " ")
// 	}
// 	// Ejecicion de comando leido
// 	return ejecucion_comando(commandArray)
// }

func ejecucion_comando(commandArray []string) string {
	respuesta := ""
	// Identificacion de comando y ejecucion
	data := strings.ToLower(commandArray[0])
	if data == "mkdisk" {
		//comando_mkdisk(commandArray)
		respuesta = comando_mkdisk(commandArray)
	} else if data == "rmdisk" {
		//comando_rmdisk(commandArray)
		respuesta = comando_rmdisk(commandArray)
	} else if data == "fdisk" {
		//comando_fkdisk(commandArray)
		respuesta = comando_fkdisk(commandArray)
	} else if data == "mount" {
		//comando_mount(commandArray)
		respuesta = comando_mount(commandArray)
	} else if data == "mkfs" {
		//comando_mkfs(commandArray)
		respuesta = comando_mkfs(commandArray)
	} else if data == "login" {
		//comando_login(commandArray)
		respuesta = comando_login(commandArray)
	} else if data == "logout" {
		//comando_logout()
		respuesta = comando_logout()
	} else if data == "mkgrp" {
		//comando_mkgrp(commandArray)
		respuesta = comando_mkgrp(commandArray)
	} else if data == "rmgrp" {
		//comando_rmgrp(commandArray)
		respuesta = comando_rmgrp(commandArray)
	} else if data == "mkusr" {
		//comando_mkusr(commandArray)
		respuesta = comando_mkusr(commandArray)
	} else if data == "rmusr" {
		//comando_rmusr(commandArray)
		respuesta = comando_rmusr(commandArray)
	} else if data == "rep" {
		//comando_rep(commandArray)
		respuesta = comando_rep(commandArray)
	} else if data == "execute" {
		comando_execute(commandArray)
	} else {
		fmt.Println("Comando ingresado no es valido")
	}
	return respuesta
}

func comando_mkdisk(commandArray []string) string {
	respuesta := ""
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
		respuesta = "¡¡ Error !! No se ha especificado una ruta para crear el disco"
		return respuesta
	}

	if stamano == "" {
		fmt.Println("¡¡ Error !! No se ha especificado el tamanio del disco")
		fmt.Println("")
		respuesta = "¡¡ Error !! No se ha especificado el tamanio del disco"
		return respuesta
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
		respuesta = "¡¡ Error !! No se pudo acceder al disco"
		return respuesta
	}
	defer file.Close()

	if _, err := file.Seek(int64(0), 0); err != nil {
		fmt.Println(err)
		//return
	}
	if err := binary.Write(file, binary.LittleEndian, &Disco); err != nil {
		fmt.Println(err)
		//return
	}

	disco.Close()

	fmt.Println("")
	fmt.Println("*                 Disco creado con exito                   *")
	fmt.Println("")

	respuesta = "*                 Disco creado con exito                   *"
	return respuesta
}

func comando_rmdisk(commandArray []string) string {
	respuesta := ""
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
		respuesta = "¡¡ Error !! No se ha especificado una ruta para eliminar"
		return respuesta
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
			fmt.Println("*               Disco eliminado con exito                  *")
			fmt.Println("")

		}

		return nil
	})

	if err != nil {
		fmt.Println("¡¡ Error !! No se ha encontrado el archivo", err)
		respuesta = "¡¡ Error !! No se ha encontrado el archivo"
		return respuesta
	}

	respuesta = "*               Disco eliminado con exito                  *"
	return respuesta

}

/*

*? Faltan las particiones logicas

 */

func comando_fkdisk(commandArray []string) string {
	respuesta := ""
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
		respuesta = "¡¡ Error !! No se ha especificado un tamano para la particion"
		return respuesta
	}

	if rutaa == "" {
		fmt.Println("¡¡ Error !! No se ha especificado una ruta para el disco")
		fmt.Println("")
		respuesta = "¡¡ Error !! No se ha especificado una ruta para el disco"
		return respuesta
	}

	if nombre_part == "" {
		fmt.Println("¡¡ Error !! No se ha especificado un nombre para la particion")
		fmt.Println("")
		respuesta = "¡¡ Error !! No se ha especificado un nombre para la particion"
		return respuesta
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
		respuesta = "¡¡ Error !! No se pudo acceder al disco"
		return respuesta
	}
	defer file.Close()

	if _, err := file.Seek(int64(0), 0); err != nil {
		fmt.Println(err)
		return err.Error()
	}

	var PurbeaD MBR
	if err := binary.Read(file, binary.LittleEndian, &PurbeaD); err != nil {
		fmt.Println(err)
		return err.Error()
	}

	fsType := string(bytes.TrimRight(PurbeaD.Dsk_fit[:], string(0)))
	fsType1 := string(bytes.TrimRight(PurbeaD.Mbr_dsk_signature[:], string(0)))
	fsType2 := string(bytes.TrimRight(PurbeaD.Mbr_fecha_creacion[:], string(0)))
	string_tamano := string(bytes.TrimRight(PurbeaD.Mbr_tamano[:], string(0)))
	fmt.Println("AJuste", fsType)
	fmt.Println("dsk", fsType1)
	fmt.Println("Fecha", fsType2)
	fmt.Println("Tam : ", string_tamano)

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
						respuesta = "¡¡ Error !! Ya existe una particion extendida"
						return respuesta
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
					respuesta = "¡¡ Error !! Ya existe una particion con ese nombre"
					return respuesta
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
					file, err := os.OpenFile(rutaa, os.O_RDWR, 0644)
					if err != nil {
						fmt.Println("¡¡ Error !! No se pudo acceder al disco")
						respuesta = "¡¡ Error !! No se pudo acceder al disco"
						return respuesta
					}
					defer file.Close()

					if _, err := file.Seek(int64(0), 0); err != nil {
						fmt.Println(err)
						return err.Error()
					}
					if err := binary.Write(file, binary.LittleEndian, &discop); err != nil {
						fmt.Println(err)
						return err.Error()
					}

					fmt.Println("")
					fmt.Println("*                  Particion 1 asignada                       *")
					fmt.Println("")

					respuesta = "*                  Particion 1 asignada                       *"

				} else {
					fmt.Println("¡¡ Error !! El tamano de la particion es mayor al disponible del disco")
					fmt.Println(tamano)
					fmt.Println(tamano_archivo1)
					respuesta = "¡¡ Error !! El tamano de la particion es mayor al disponible del disco"
					return respuesta
				}

				// 	tamanoDisponible = disco.mbr_tamano - disco.mbr_partition_1.part_s;
			} else if status_part1 == "1" && status_part2 == "0" && status_part3 == "0" && status_part4 == "0" {
				partSizeStr := string(particion[0].Part_size[:])
				partSizeStr = strings.TrimRightFunc(partSizeStr, func(r rune) bool { return r == '\x00' })
				partSizeInt, err := strconv.Atoi(partSizeStr)
				//pruebainicio := (partSizeInt * 1024)
				pruebainicio := (partSizeInt)
				fmt.Println("Que sale en bytes ", pruebainicio)
				if err != nil {
					// Manejo del error
				}
				tamanoDisponibleAntes1 := tamano - partSizeInt
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

					//Apertura del archivo
					file, err := os.OpenFile(rutaa, os.O_RDWR, 0644)
					if err != nil {
						fmt.Println("¡¡ Error !! No se pudo acceder al disco")
						respuesta = "¡¡ Error !! No se pudo acceder al disco"
						return respuesta
					}
					defer file.Close()

					if _, err := file.Seek(int64(0), 0); err != nil {
						fmt.Println(err)
						return err.Error()
					}
					if err := binary.Write(file, binary.LittleEndian, &discop); err != nil {
						fmt.Println(err)
						return err.Error()
					}

					fmt.Println("")
					fmt.Println("*                  Particion 2 asignada                       *")
					fmt.Println("")
					respuesta = "*                  Particion 2 asignada                       *"
				} else {
					fmt.Println("¡¡ Error !! El tamano de la particion es mayor al disponible del disco")
					fmt.Println(tamano)
					fmt.Println(tamano_archivo1)
					respuesta = "¡¡ Error !! El tamano de la particion es mayor al disponible del disco"
					return respuesta
				}

			} else if status_part1 == "1" && status_part2 == "1" && status_part3 == "0" && status_part4 == "0" {
				partSizeStr := string(particion[0].Part_size[:])
				partSizeStr1 := string(particion[1].Part_size[:])
				partSizeStr = strings.TrimRightFunc(partSizeStr, func(r rune) bool { return r == '\x00' })
				partSizeStr1 = strings.TrimRightFunc(partSizeStr1, func(r rune) bool { return r == '\x00' })
				partSizeInt, err := strconv.Atoi(partSizeStr)
				partSizeInt1, err := strconv.Atoi(partSizeStr1)
				//pruebainicio := (partSizeInt1 * 1024)
				pruebainicio := (partSizeInt1)
				if err != nil {
					// Manejo del error
				}
				tamanoDisponibleAntes1 := tamano - (partSizeInt + partSizeInt1)
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
					copy(discop.Mbr_partition_3.Part_start[:], strconv.Itoa(startt))
					copy(discop.Mbr_partition_3.Part_size[:], strconv.Itoa(tamano_archivo1))
					copy(discop.Mbr_partition_3.Part_name[:], nombre_part)

					//Apertura del archivo
					file, err := os.OpenFile(rutaa, os.O_RDWR, 0644)
					if err != nil {
						fmt.Println("¡¡ Error !! No se pudo acceder al disco")
						respuesta = "¡¡ Error !! No se pudo acceder al disco"
						return respuesta
					}
					defer file.Close()

					if _, err := file.Seek(int64(0), 0); err != nil {
						fmt.Println(err)
						return err.Error()
					}
					if err := binary.Write(file, binary.LittleEndian, &discop); err != nil {
						fmt.Println(err)
						return err.Error()
					}

					fmt.Println("")
					fmt.Println("*                  Particion 3 asignada                       *")
					fmt.Println("")
					respuesta = "*                  Particion 3 asignada                       *"
				} else {
					fmt.Println("¡¡ Error !! El tamano de la particion es mayor al disponible del disco")
					fmt.Println(tamano)
					fmt.Println(tamano_archivo1)
					respuesta = "¡¡ Error !! El tamano de la particion es mayor al disponible del disco"
					return respuesta
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
				//pruebainicio := (partSizeInt2 * 1024)
				pruebainicio := (partSizeInt2)
				if err != nil {
					// Manejo del error
				}
				tamanoDisponibleAntes1 := tamano - (partSizeInt + partSizeInt1 + partSizeInt2)
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
					copy(discop.Mbr_partition_4.Part_start[:], strconv.Itoa(startt))
					copy(discop.Mbr_partition_4.Part_size[:], strconv.Itoa(tamano_archivo1))
					copy(discop.Mbr_partition_4.Part_name[:], nombre_part)

					//Apertura del archivo
					file, err := os.OpenFile(rutaa, os.O_RDWR, 0644)
					if err != nil {
						fmt.Println("¡¡ Error !! No se pudo acceder al disco")
						respuesta = "¡¡ Error !! No se pudo acceder al disco"
						return respuesta
					}
					defer file.Close()

					if _, err := file.Seek(int64(0), 0); err != nil {
						fmt.Println(err)
						return err.Error()
					}
					if err := binary.Write(file, binary.LittleEndian, &discop); err != nil {
						fmt.Println(err)
						return err.Error()
					}
					fmt.Println("")
					fmt.Println("*                  Particion 4 asignada                       *")
					fmt.Println("")
					respuesta = "*                  Particion 4 asignada                       *"
				} else {
					fmt.Println("¡¡ Error !! El tamano de la particion es mayor al disponible del disco")
					fmt.Println(tamano)
					fmt.Println(tamano_archivo1)
					respuesta = "¡¡ Error !! El tamano de la particion es mayor al disponible del disco"
					return respuesta
				}
			} else if status_part1 == "1" && status_part2 == "1" && status_part3 == "1" && status_part4 == "1" {
				fmt.Println("¡¡ Error !! Ya no hay particiones disponibles")
				respuesta = "¡¡ Error !! Ya no hay particiones disponibles"
				return respuesta
			}
		} else if part_type == "L" {
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
				respuesta = "¡¡ Error !! Primero debe crear una particion Extendida"
				return respuesta
			}
			//}

			if encontrado == true {

				fmt.Println("Llega ?")
				inicio_particion1 := string(particion[0].Part_start[:])
				inicio_particion1 = strings.TrimRightFunc(inicio_particion1, func(r rune) bool { return r == '\x00' })
				size_particion1 := string(particion[0].Part_size[:])
				size_particion1 = strings.TrimRightFunc(size_particion1, func(r rune) bool { return r == '\x00' })
				//int_inicio_particion1, err := strconv.Atoi(inicio_particion1)
				int_inicio_particion1, err := strconv.Atoi(inicio_particion1)
				int_size_particion1, err := strconv.Atoi(size_particion1)
				file, err := os.OpenFile(rutaa, os.O_RDONLY, 0644)
				if err != nil {
					fmt.Println("¡¡ Error !! No se pudo acceder al disco")
					respuesta = "¡¡ Error !! No se pudo acceder al disco"
					return respuesta
				}
				defer file.Close()

				if _, err := file.Seek(int64(int_inicio_particion1), 0); err != nil {
					fmt.Println(err)
					return err.Error()
				}

				var PruebaEBR EBR
				if err := binary.Read(file, binary.LittleEndian, &PruebaEBR); err != nil {
					fmt.Println(err)
					return err.Error()
				}

				// fsType := string(bytes.TrimRight(PruebaEBR.Part_fit[:], string(0)))
				fsType1 := string(bytes.TrimRight(PruebaEBR.Part_name[:], string(0)))
				// fsType2 := string(bytes.TrimRight(PruebaEBR.Part_next[:], string(0)))
				// string_tamanop := string(bytes.TrimRight(PruebaEBR.Part_size[:], string(0)))
				// int_tam_disco, err := strconv.Atoi(string_tamanop)
				// fmt.Println("AJuste", fsType)
				// fmt.Println("Nombre", fsType1)
				// fmt.Println("Siguiente", fsType2)
				// fmt.Println("Tam : ", string_tamanop)

				if fsType1 == "" {

					if int_size_particion1 >= tamano_archivo1 {

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
							respuesta = "¡¡ Error !! No se pudo acceder al disco"
							return respuesta
						}
						defer file.Close()

						if _, err := file.Seek(int64(int_inicio_particion1), 0); err != nil {
							fmt.Println(err)
							return err.Error()
						}
						if err := binary.Write(file, binary.LittleEndian, &obtener_ebr); err != nil {
							fmt.Println(err)
							return err.Error()
						}

						fmt.Println("")
						fmt.Println("*                Particion Logica creada                      *")
						fmt.Println("")
						respuesta = "*                Particion Logica creada                      *"

						return respuesta

					} else {
						fmt.Println("¡¡ Error !! El tamano de la particion Logica supera al tamano de la extendida")
						respuesta = "¡¡ Error !! El tamano de la particion Logica supera al tamano de la extendida"
						return respuesta
					}
				} else {
					fmt.Println("Entra el otro")

					var AnteriorEBR EBR
					var SiguienteEBR EBR

					AnteriorEBR = PruebaEBR
					nombre_anterior := string(bytes.TrimRight(PruebaEBR.Part_name[:], string(0)))
					size_anterior := string(bytes.TrimRight(PruebaEBR.Part_size[:], string(0)))

					int_anterior, err := strconv.Atoi(size_anterior)
					if err != nil {

					}

					if nombre_part == nombre_anterior {
						fmt.Println("¡¡ Error !! Ya está en uso ese nombre en una particion")
						respuesta = "¡¡ Error !! Ya está en uso ese nombre en una particion"
						return respuesta
					} else {
						tamanofinal := int_size_particion1 - (int_anterior)
						nuevostart := int_inicio_particion1 + int_anterior + 1

						fmt.Println("? ", tamanofinal)

						if tamanofinal >= tamano_archivo1 {

							copy(AnteriorEBR.Part_next[:], []byte(strconv.Itoa(nuevostart)))

							copy(SiguienteEBR.Part_status[:], "0")
							copy(SiguienteEBR.Part_fit[:], part_fit)
							copy(SiguienteEBR.Part_start[:], strconv.Itoa(nuevostart))
							copy(SiguienteEBR.Part_size[:], strconv.Itoa(tamano_archivo1))
							copy(SiguienteEBR.Part_next[:], strconv.Itoa(-1))
							copy(SiguienteEBR.Part_name[:], nombre_part)

							file, err := os.OpenFile(rutaa, os.O_RDWR, 0644)
							if err != nil {
								fmt.Println("¡¡ Error !! No se pudo acceder al disco")
								respuesta = "¡¡ Error !! No se pudo acceder al disco"
								return respuesta
							}
							defer file.Close()

							if _, err := file.Seek(int64(int_inicio_particion1), 0); err != nil {
								fmt.Println(err)
								return err.Error()
							}
							if err := binary.Write(file, binary.LittleEndian, &AnteriorEBR); err != nil {
								fmt.Println(err)
								return err.Error()
							}
							if err := binary.Write(file, binary.LittleEndian, &SiguienteEBR); err != nil {
								fmt.Println(err)
								return err.Error()
							}

							fmt.Println("")
							fmt.Println("*                Particion Logica creada                      *")
							fmt.Println("")
							respuesta = "*                Particion Logica creada                      *"

							return respuesta

						} else {
							fmt.Println("¡¡ Error !!")
							respuesta = "¡¡ Error !!"
							return respuesta
						}

					}

				}

			} else if encontrado1 == true {
				fmt.Println("Entra al if encontrado 2")

				inicio_particion2 := string(particion[1].Part_start[:])
				inicio_particion2 = strings.TrimRightFunc(inicio_particion2, func(r rune) bool { return r == '\x00' })
				size_particion2 := string(particion[1].Part_size[:])
				size_particion2 = strings.TrimRightFunc(size_particion2, func(r rune) bool { return r == '\x00' })
				//int_inicio_particion1, err := strconv.Atoi(inicio_particion1)
				int_inicio_particion2, err := strconv.Atoi(inicio_particion2)
				int_size_particion2, err := strconv.Atoi(size_particion2)
				file, err := os.OpenFile(rutaa, os.O_RDONLY, 0644)
				if err != nil {
					fmt.Println("¡¡ Error !! No se pudo acceder al disco")
					respuesta = "¡¡ Error !! No se pudo acceder al disco"
					return respuesta
				}
				defer file.Close()

				if _, err := file.Seek(int64(int_inicio_particion2), 0); err != nil {
					fmt.Println(err)
					return err.Error()
				}

				var PruebaEBR EBR
				if err := binary.Read(file, binary.LittleEndian, &PruebaEBR); err != nil {
					fmt.Println(err)
					return err.Error()
				}

				fsType2 := string(bytes.TrimRight(PruebaEBR.Part_name[:], string(0)))

				if fsType2 == "" {

					if int_size_particion2 >= tamano_archivo1 {

						inicio_particion2 := string(particion[1].Part_start[:])
						inicio_particion2 = strings.TrimRightFunc(inicio_particion2, func(r rune) bool { return r == '\x00' })
						//int_inicio_particion1, err := strconv.Atoi(inicio_particion1)
						int_inicio_particion2, err := strconv.Atoi(inicio_particion2)

						obtener_ebr := PruebaEBR

						copy(obtener_ebr.Part_status[:], "0")
						copy(obtener_ebr.Part_fit[:], part_fit)
						copy(obtener_ebr.Part_start[:], strconv.Itoa(int_inicio_particion2))
						copy(obtener_ebr.Part_size[:], strconv.Itoa(tamano_archivo1))
						copy(obtener_ebr.Part_next[:], strconv.Itoa(-1))
						copy(obtener_ebr.Part_name[:], nombre_part)

						file, err := os.OpenFile(rutaa, os.O_RDWR, 0644)
						if err != nil {
							fmt.Println("¡¡ Error !! No se pudo acceder al disco")
							respuesta = "¡¡ Error !! No se pudo acceder al disco"
							return respuesta
						}
						defer file.Close()

						if _, err := file.Seek(int64(int_inicio_particion2), 0); err != nil {
							fmt.Println(err)
							return err.Error()
						}
						if err := binary.Write(file, binary.LittleEndian, &obtener_ebr); err != nil {
							fmt.Println(err)
							return err.Error()
						}

						fmt.Println("")
						fmt.Println("*                Particion Logica creada                      *")
						fmt.Println("")
						respuesta = "*                Particion Logica creada                      *"

						return respuesta

					} else {
						fmt.Println("¡¡ Error !! El tamano de la particion Logica supera al tamano de la extendida")
						respuesta = "¡¡ Error !! El tamano de la particion Logica supera al tamano de la extendida"
						return respuesta
					}
				} else {

					var AnteriorEBR EBR
					var SiguienteEBR EBR

					AnteriorEBR = PruebaEBR
					nombre_anterior := string(bytes.TrimRight(PruebaEBR.Part_name[:], string(0)))
					size_anterior := string(bytes.TrimRight(PruebaEBR.Part_size[:], string(0)))

					int_anterior, err := strconv.Atoi(size_anterior)
					if err != nil {

					}

					if nombre_part == nombre_anterior {
						fmt.Println("¡¡ Error !! Ya está en uso ese nombre en una particion")
						respuesta = "¡¡ Error !! Ya está en uso ese nombre en una particion"
						return respuesta
					} else {
						tamanofinal := int_size_particion2 - (int_anterior)
						nuevostart := int_inicio_particion2 + int_anterior + 1

						if tamanofinal >= tamano_archivo1 {

							copy(AnteriorEBR.Part_next[:], []byte(strconv.Itoa(nuevostart)))

							copy(SiguienteEBR.Part_status[:], "0")
							copy(SiguienteEBR.Part_fit[:], part_fit)
							copy(SiguienteEBR.Part_start[:], strconv.Itoa(nuevostart))
							copy(SiguienteEBR.Part_size[:], strconv.Itoa(tamano_archivo1))
							copy(SiguienteEBR.Part_next[:], strconv.Itoa(-1))
							copy(SiguienteEBR.Part_name[:], nombre_part)

							file, err := os.OpenFile(rutaa, os.O_RDWR, 0644)
							if err != nil {
								fmt.Println("¡¡ Error !! No se pudo acceder al disco")
								respuesta = "¡¡ Error !! No se pudo acceder al disco"
								return respuesta
							}
							defer file.Close()

							if _, err := file.Seek(int64(int_inicio_particion2), 0); err != nil {
								fmt.Println(err)
								return err.Error()
							}
							if err := binary.Write(file, binary.LittleEndian, &AnteriorEBR); err != nil {
								fmt.Println(err)
								return err.Error()
							}
							if err := binary.Write(file, binary.LittleEndian, &SiguienteEBR); err != nil {
								fmt.Println(err)
								return err.Error()
							}

						} else {
							fmt.Println("")
							fmt.Println("*                Particion Logica creada                      *")
							fmt.Println("")
							respuesta = "*                Particion Logica creada                      *"

							return respuesta
						}

					}

				}

			} else if encontrado2 == true {
				fmt.Println("Entra al if encontrado 3")

				inicio_particion3 := string(particion[2].Part_start[:])
				inicio_particion3 = strings.TrimRightFunc(inicio_particion3, func(r rune) bool { return r == '\x00' })
				size_particion3 := string(particion[2].Part_size[:])
				size_particion3 = strings.TrimRightFunc(size_particion3, func(r rune) bool { return r == '\x00' })
				//int_inicio_particion1, err := strconv.Atoi(inicio_particion1)
				int_inicio_particion3, err := strconv.Atoi(inicio_particion3)
				int_size_particion3, err := strconv.Atoi(size_particion3)
				file, err := os.OpenFile(rutaa, os.O_RDONLY, 0644)
				if err != nil {
					fmt.Println("¡¡ Error !! No se pudo acceder al disco")
					respuesta = "¡¡ Error !! No se pudo acceder al disco"
					return respuesta
				}
				defer file.Close()

				if _, err := file.Seek(int64(int_inicio_particion3), 0); err != nil {
					fmt.Println(err)
					return err.Error()
				}

				var PruebaEBR EBR
				if err := binary.Read(file, binary.LittleEndian, &PruebaEBR); err != nil {
					fmt.Println(err)
					return err.Error()
				}

				fsType3 := string(bytes.TrimRight(PruebaEBR.Part_name[:], string(0)))

				if fsType3 == "" {

					if int_size_particion3 >= tamano_archivo1 {

						inicio_particion3 := string(particion[2].Part_start[:])
						inicio_particion3 = strings.TrimRightFunc(inicio_particion3, func(r rune) bool { return r == '\x00' })
						//int_inicio_particion1, err := strconv.Atoi(inicio_particion1)
						int_inicio_particion3, err := strconv.Atoi(inicio_particion3)

						obtener_ebr := PruebaEBR

						copy(obtener_ebr.Part_status[:], "0")
						copy(obtener_ebr.Part_fit[:], part_fit)
						copy(obtener_ebr.Part_start[:], strconv.Itoa(int_inicio_particion3))
						copy(obtener_ebr.Part_size[:], strconv.Itoa(tamano_archivo1))
						copy(obtener_ebr.Part_next[:], strconv.Itoa(-1))
						copy(obtener_ebr.Part_name[:], nombre_part)

						file, err := os.OpenFile(rutaa, os.O_RDWR, 0644)
						if err != nil {
							fmt.Println("¡¡ Error !! No se pudo acceder al disco")
							respuesta = "¡¡ Error !! No se pudo acceder al disco"
							return respuesta
						}
						defer file.Close()

						if _, err := file.Seek(int64(int_inicio_particion3), 0); err != nil {
							fmt.Println(err)
							return err.Error()
						}
						if err := binary.Write(file, binary.LittleEndian, &obtener_ebr); err != nil {
							fmt.Println(err)
							return err.Error()
						}

						fmt.Println("")
						fmt.Println("*                Particion Logica creada                      *")
						fmt.Println("")
						respuesta = "*                Particion Logica creada                      *"

						return respuesta

					} else {
						fmt.Println("¡¡ Error !! El tamano de la particion Logica supera al tamano de la extendida")
						respuesta = "¡¡ Error !! El tamano de la particion Logica supera al tamano de la extendida"
						return respuesta
					}
				} else {

					var AnteriorEBR EBR
					var SiguienteEBR EBR

					AnteriorEBR = PruebaEBR
					nombre_anterior := string(bytes.TrimRight(PruebaEBR.Part_name[:], string(0)))
					size_anterior := string(bytes.TrimRight(PruebaEBR.Part_size[:], string(0)))

					int_anterior, err := strconv.Atoi(size_anterior)
					if err != nil {

					}

					if nombre_part == nombre_anterior {
						fmt.Println("¡¡ Error !! Ya está en uso ese nombre en una particion")
						respuesta = "¡¡ Error !! Ya está en uso ese nombre en una particion"
						return respuesta
					} else {
						tamanofinal := int_size_particion3 - (int_anterior)
						nuevostart := int_inicio_particion3 + int_anterior + 1

						if tamanofinal >= tamano_archivo1 {

							copy(AnteriorEBR.Part_next[:], []byte(strconv.Itoa(nuevostart)))

							copy(SiguienteEBR.Part_status[:], "0")
							copy(SiguienteEBR.Part_fit[:], part_fit)
							copy(SiguienteEBR.Part_start[:], strconv.Itoa(nuevostart))
							copy(SiguienteEBR.Part_size[:], strconv.Itoa(tamano_archivo1))
							copy(SiguienteEBR.Part_next[:], strconv.Itoa(-1))
							copy(SiguienteEBR.Part_name[:], nombre_part)

							file, err := os.OpenFile(rutaa, os.O_RDWR, 0644)
							if err != nil {
								fmt.Println("¡¡ Error !! No se pudo acceder al disco")
								respuesta = "¡¡ Error !! No se pudo acceder al disco"
								return respuesta
							}
							defer file.Close()

							if _, err := file.Seek(int64(int_inicio_particion3), 0); err != nil {
								fmt.Println(err)
								return err.Error()
							}
							if err := binary.Write(file, binary.LittleEndian, &AnteriorEBR); err != nil {
								fmt.Println(err)
								return err.Error()
							}
							if err := binary.Write(file, binary.LittleEndian, &SiguienteEBR); err != nil {
								fmt.Println(err)
								return err.Error()
							}

						} else {
							fmt.Println("")
							fmt.Println("*                Particion Logica creada                      *")
							fmt.Println("")
							respuesta = "*                Particion Logica creada                      *"

							return respuesta
						}

					}

				}

			} else if encontrado3 == true {
				fmt.Println("Entra al if encontrado 4")

				inicio_particion4 := string(particion[3].Part_start[:])
				inicio_particion4 = strings.TrimRightFunc(inicio_particion4, func(r rune) bool { return r == '\x00' })
				size_particion4 := string(particion[3].Part_size[:])
				size_particion4 = strings.TrimRightFunc(size_particion4, func(r rune) bool { return r == '\x00' })
				//int_inicio_particion1, err := strconv.Atoi(inicio_particion1)
				int_inicio_particion4, err := strconv.Atoi(inicio_particion4)
				int_size_particion4, err := strconv.Atoi(size_particion4)
				file, err := os.OpenFile(rutaa, os.O_RDONLY, 0644)
				if err != nil {
					fmt.Println("¡¡ Error !! No se pudo acceder al disco")
					respuesta = "¡¡ Error !! No se pudo acceder al disco"
					return respuesta
				}
				defer file.Close()

				if _, err := file.Seek(int64(int_inicio_particion4), 0); err != nil {
					fmt.Println(err)
					return err.Error()
				}

				var PruebaEBR EBR
				if err := binary.Read(file, binary.LittleEndian, &PruebaEBR); err != nil {
					fmt.Println(err)
					return err.Error()
				}

				fsType4 := string(bytes.TrimRight(PruebaEBR.Part_name[:], string(0)))

				if fsType4 == "" {

					if int_size_particion4 >= tamano_archivo1 {

						inicio_particion4 := string(particion[3].Part_start[:])
						inicio_particion4 = strings.TrimRightFunc(inicio_particion4, func(r rune) bool { return r == '\x00' })
						//int_inicio_particion1, err := strconv.Atoi(inicio_particion1)
						int_inicio_particion4, err := strconv.Atoi(inicio_particion4)

						obtener_ebr := PruebaEBR

						copy(obtener_ebr.Part_status[:], "0")
						copy(obtener_ebr.Part_fit[:], part_fit)
						copy(obtener_ebr.Part_start[:], strconv.Itoa(int_inicio_particion4))
						copy(obtener_ebr.Part_size[:], strconv.Itoa(tamano_archivo1))
						copy(obtener_ebr.Part_next[:], strconv.Itoa(-1))
						copy(obtener_ebr.Part_name[:], nombre_part)

						file, err := os.OpenFile(rutaa, os.O_RDWR, 0644)
						if err != nil {
							fmt.Println("¡¡ Error !! No se pudo acceder al disco")
							respuesta = "¡¡ Error !! No se pudo acceder al disco"
							return respuesta
						}
						defer file.Close()

						if _, err := file.Seek(int64(int_inicio_particion4), 0); err != nil {
							fmt.Println(err)
							return err.Error()
						}
						if err := binary.Write(file, binary.LittleEndian, &obtener_ebr); err != nil {
							fmt.Println(err)
							return err.Error()
						}

						fmt.Println("")
						fmt.Println("*                Particion Logica creada                      *")
						fmt.Println("")
						respuesta = "*                Particion Logica creada                      *"

						return respuesta

					} else {
						fmt.Println("¡¡ Error !! El tamano de la particion Logica supera al tamano de la extendida")
						respuesta = "¡¡ Error !! El tamano de la particion Logica supera al tamano de la extendida"
						return respuesta
					}
				} else {

					var AnteriorEBR EBR
					var SiguienteEBR EBR

					AnteriorEBR = PruebaEBR
					nombre_anterior := string(bytes.TrimRight(PruebaEBR.Part_name[:], string(0)))
					size_anterior := string(bytes.TrimRight(PruebaEBR.Part_size[:], string(0)))

					int_anterior, err := strconv.Atoi(size_anterior)
					if err != nil {

					}

					if nombre_part == nombre_anterior {
						fmt.Println("¡¡ Error !! Ya está en uso ese nombre en una particion")
						respuesta = "¡¡ Error !! Ya está en uso ese nombre en una particion"
						return respuesta
					} else {
						tamanofinal := int_size_particion4 - (int_anterior)
						nuevostart := int_inicio_particion4 + int_anterior + 1

						if tamanofinal >= tamano_archivo1 {

							copy(AnteriorEBR.Part_next[:], []byte(strconv.Itoa(nuevostart)))

							copy(SiguienteEBR.Part_status[:], "0")
							copy(SiguienteEBR.Part_fit[:], part_fit)
							copy(SiguienteEBR.Part_start[:], strconv.Itoa(nuevostart))
							copy(SiguienteEBR.Part_size[:], strconv.Itoa(tamano_archivo1))
							copy(SiguienteEBR.Part_next[:], strconv.Itoa(-1))
							copy(SiguienteEBR.Part_name[:], nombre_part)

							file, err := os.OpenFile(rutaa, os.O_RDWR, 0644)
							if err != nil {
								fmt.Println("¡¡ Error !! No se pudo acceder al disco")
								respuesta = "¡¡ Error !! No se pudo acceder al disco"
								return respuesta
							}
							defer file.Close()

							if _, err := file.Seek(int64(int_inicio_particion4), 0); err != nil {
								fmt.Println(err)
								return err.Error()
							}
							if err := binary.Write(file, binary.LittleEndian, &AnteriorEBR); err != nil {
								fmt.Println(err)
								return err.Error()
							}
							if err := binary.Write(file, binary.LittleEndian, &SiguienteEBR); err != nil {
								fmt.Println(err)
								return err.Error()
							}

						} else {
							fmt.Println("")
							fmt.Println("*                Particion Logica creada                      *")
							fmt.Println("")
							respuesta = "*                Particion Logica creada                      *"

							return respuesta
						}

					}
				}
			}
		}
	} else {
		fmt.Println("¡¡ Error !! El tamano de la particion es mayor al disponible del disco")
		fmt.Println(tamano)
		fmt.Println(tamano_archivo1)
		respuesta = "¡¡ Error !! El tamano de la particion es mayor al disponible del disco"
		return respuesta
	}

	return respuesta
}

func comando_mount(commandArray []string) string {
	respuesta := ""
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
		respuesta = "¡¡ Error !! No se ha especificado una ruta para el disco"
		return respuesta
	}

	if nombre_part == "" {
		fmt.Println("¡¡ Error !! No se ha especificado un nombre para la particion")
		fmt.Println("")
		respuesta = "¡¡ Error !! No se ha especificado un nombre para la particion"
		return respuesta
	}

	file, err := os.OpenFile(rutaa, os.O_RDONLY, 0644)
	if err != nil {
		fmt.Println("¡¡ Error !! No se pudo acceder al disco")
		respuesta = "¡¡ Error !! No se pudo acceder al disco"
		return respuesta
	}
	defer file.Close()

	if _, err := file.Seek(int64(0), 0); err != nil {
		fmt.Println(err)
		return err.Error()
	}

	var PurbeaD MBR
	if err := binary.Read(file, binary.LittleEndian, &PurbeaD); err != nil {
		fmt.Println(err)
		return err.Error()
	}

	fsType := string(bytes.TrimRight(PurbeaD.Dsk_fit[:], string(0)))
	fsType1 := string(bytes.TrimRight(PurbeaD.Mbr_dsk_signature[:], string(0)))
	fsType2 := string(bytes.TrimRight(PurbeaD.Mbr_fecha_creacion[:], string(0)))
	string_tamano := string(bytes.TrimRight(PurbeaD.Mbr_tamano[:], string(0)))
	fmt.Println("AJuste", fsType)
	fmt.Println("dsk", fsType1)
	fmt.Println("Fecha", fsType2)
	fmt.Println("Tam : ", string_tamano)

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

	if name_part1 == nombre_part {

		montado := string(discop.Mbr_partition_1.Part_status[:])
		montado = strings.TrimRightFunc(montado, func(r rune) bool { return r == '\x00' })

		if montado == "1" {
			fmt.Println("¡¡ Error !! La particion ya se encuentra montada")
			respuesta = "¡¡ Error !! La particion ya se encuentra montada"
			return respuesta
		}

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

		file, err := os.OpenFile(rutaa, os.O_RDWR, 0644)
		if err != nil {
			fmt.Println("¡¡ Error !! No se pudo acceder al disco")
			respuesta = "¡¡ Error !! No se pudo acceder al disco"
			return respuesta
		}
		defer file.Close()

		if _, err := file.Seek(int64(0), 0); err != nil {
			fmt.Println(err)
			return err.Error()
		}
		if err := binary.Write(file, binary.LittleEndian, &discop); err != nil {
			fmt.Println(err)
			return err.Error()
		}

		fmt.Println("")
		fmt.Println("*               Particion montada con exito                *")
		fmt.Println("")
		respuesta = "*               Particion montada con exito                *"

		miLista.ImprimirTabla()
	} else if name_part2 == nombre_part {
		montado := string(discop.Mbr_partition_2.Part_status[:])
		montado = strings.TrimRightFunc(montado, func(r rune) bool { return r == '\x00' })

		if montado == "1" {
			fmt.Println("¡¡ Error !! La particion ya se encuentra montada")
			respuesta = "¡¡ Error !! La particion ya se encuentra montada"
			return respuesta
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

		file, err := os.OpenFile(rutaa, os.O_RDWR, 0644)
		if err != nil {
			fmt.Println("¡¡ Error !! No se pudo acceder al disco")
			respuesta = "¡¡ Error !! No se pudo acceder al disco"
			return respuesta
		}
		defer file.Close()

		if _, err := file.Seek(int64(0), 0); err != nil {
			fmt.Println(err)
			return err.Error()
		}
		if err := binary.Write(file, binary.LittleEndian, &discop); err != nil {
			fmt.Println(err)
			return err.Error()
		}

		fmt.Println("")
		fmt.Println("*               Particion montada con exito                *")
		fmt.Println("")
		respuesta = "*               Particion montada con exito                *"

		miLista.ImprimirTabla()
	} else if name_part3 == nombre_part {
		montado := string(discop.Mbr_partition_3.Part_status[:])
		montado = strings.TrimRightFunc(montado, func(r rune) bool { return r == '\x00' })

		if montado == "1" {
			fmt.Println("¡¡ Error !! La particion ya se encuentra montada")
			respuesta = "¡¡ Error !! La particion ya se encuentra montada"
			return respuesta
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

		file, err := os.OpenFile(rutaa, os.O_RDWR, 0644)
		if err != nil {
			fmt.Println("¡¡ Error !! No se pudo acceder al disco")
			respuesta = "¡¡ Error !! No se pudo acceder al disco"
			return respuesta
		}
		defer file.Close()

		if _, err := file.Seek(int64(0), 0); err != nil {
			fmt.Println(err)
			return err.Error()
		}
		if err := binary.Write(file, binary.LittleEndian, &discop); err != nil {
			fmt.Println(err)
			return err.Error()
		}

		fmt.Println("")
		fmt.Println("*               Particion montada con exito                *")
		fmt.Println("")
		respuesta = "*               Particion montada con exito                *"

		miLista.ImprimirTabla()
	} else if name_part4 == nombre_part {
		montado := string(discop.Mbr_partition_4.Part_status[:])
		montado = strings.TrimRightFunc(montado, func(r rune) bool { return r == '\x00' })

		if montado == "1" {
			fmt.Println("¡¡ Error !! La particion ya se encuentra montada")
			respuesta = "¡¡ Error !! La particion ya se encuentra montada"
			return respuesta
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

		file, err := os.OpenFile(rutaa, os.O_RDWR, 0644)
		if err != nil {
			fmt.Println("¡¡ Error !! No se pudo acceder al disco")
			respuesta = "¡¡ Error !! No se pudo acceder al disco"
			return respuesta
		}
		defer file.Close()

		if _, err := file.Seek(int64(0), 0); err != nil {
			fmt.Println(err)
			return err.Error()
		}
		if err := binary.Write(file, binary.LittleEndian, &discop); err != nil {
			fmt.Println(err)
			return err.Error()
		}

		fmt.Println("")
		fmt.Println("*               Particion montada con exito                *")
		fmt.Println("")
		respuesta = "*               Particion montada con exito                *"

		miLista.ImprimirTabla()
	} else {
		fmt.Println("¡¡ Error !! No se encontro una particion con ese nombre")
		respuesta = "¡¡ Error !! No se encontro una particion con ese nombre"
		return respuesta
	}

	return respuesta

}

func comando_mkfs(commandArray []string) string {
	respuesta := ""
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
		respuesta = "¡¡ Error !! No se ha especificado un id para el formateo"
		return respuesta
	}

	tipo_formateo := ""

	if type_part == "full" {
		tipo_formateo = "full"
	} else if type_part == "" {
		tipo_formateo = "full"
	} else {
		fmt.Println("El tipo de formateo no es aceptado")
		respuesta = "El tipo de formateo no es aceptado"
		return respuesta
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
			respuesta = "*               Formato EXT2 creado con exito              *"

		} else {
			fmt.Println("No se encontró ningúna particion con ese ID")
			respuesta = "No se encontró ningúna particion con ese ID"
			return respuesta
		}
	}
	//respuesta = "*               Formato EXT2 creado con exito              *"

	return respuesta
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
	//inode_star := int(binary.LittleEndian.Uint32(SP.S_firts_ino[:]))
	inode_star := first_in
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

	fecha_creacion := time.Now().Format("2006-01-02 15:04:05")
	fecha_atime := string(fecha_creacion)

	var fechaBytes [19]byte

	copy(fechaBytes[:], []byte(fecha_atime))

	// Se crean manualmente los primeros Inodos

	Predeterminado := make([]Inodos, 2)

	copy(Predeterminado[0].I_uid[:], "1")
	copy(Predeterminado[0].I_gid[:], "1")
	copy(Predeterminado[0].I_atime[:], []byte(fecha_atime))
	copy(Predeterminado[0].I_ctime[:], []byte(fecha_atime))
	copy(Predeterminado[0].I_mtime[:], []byte(fecha_atime))
	copy(Predeterminado[0].I_block[:], "0")
	copy(Predeterminado[0].I_type[:], "0")
	copy(Predeterminado[0].I_perm[:], strconv.Itoa(664))

	copy(Predeterminado[1].I_uid[:], "1")
	copy(Predeterminado[1].I_gid[:], "1")
	copy(Predeterminado[1].I_atime[:], []byte(fecha_atime))
	copy(Predeterminado[1].I_ctime[:], []byte(fecha_atime))
	copy(Predeterminado[1].I_mtime[:], []byte(fecha_atime))
	copy(Predeterminado[1].I_block[:], "1")
	copy(Predeterminado[1].I_type[:], "1")
	copy(Predeterminado[1].I_perm[:], strconv.Itoa(700))

	EscribirInodos(nodoActual.ruta, Predeterminado, int64(nodoActual.inicioparticion), n, n*3)

	Inn := Inodos{}
	Inode_size := unsafe.Sizeof(Inn)

	nuevo_first_in := first_in + 2*int(Inode_size)
	copy(SP.S_firts_ino[:], strconv.Itoa(nuevo_first_in))

	Carpeta := make([]BloqueCarpetas, 4)
	//Carpeta := BloqueCarpetas{}
	contenidoCarpeta := Content{}
	copy(contenidoCarpeta.B_inodo[:], "1")
	copy(contenidoCarpeta.B_name[:], "users.txt")

	copy(Carpeta[0].B_content[0].B_name[:], contenidoCarpeta.B_name[:])
	copy(Carpeta[0].B_content[1].B_inodo[:], []byte(strconv.Itoa(-1)))
	copy(Carpeta[0].B_content[2].B_inodo[:], []byte(strconv.Itoa(-1)))
	copy(Carpeta[0].B_content[3].B_inodo[:], []byte(strconv.Itoa(-1)))

	Escribir_BloqueCarpetas(nodoActual.ruta, Carpeta, nodoActual.inicioparticion, n)

	//Bloque Archivos
	Archivo := BloqueArchivos{}
	contenidoarchivo := "1,G,root\n1,U,root,root,123\n"
	copy(Archivo.B_content[:], []byte(contenidoarchivo))

	Escribir_BloqueArchivo(nodoActual.ruta, Archivo, nodoActual.inicioparticion, n)

	Carpet := BloqueCarpetas{}
	Carpet_size := unsafe.Sizeof(Carpet)
	Arch := BloqueArchivos{}
	Arch_size := unsafe.Sizeof(Arch)

	nuevo_first_blo := first_blo + int(Carpet_size) + int(Arch_size)
	copy(SP.S_first_blo[:], strconv.Itoa(nuevo_first_blo))

	Actualizar_SuperBloque(nodoActual.ruta, SP, nodoActual.inicioparticion)

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
		offset1 := offset + int64(prueba) + 30
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
		//fmt.Println("DOnde manda ", offset1)
		if _, err := file.Seek(offset1, 0); err != nil {
			return err
		}
		if err := binary.Write(file, binary.LittleEndian, &bmBloque[i]); err != nil {
			return err
		}
	}

	return nil
}

func EscribirInodos(path string, inodo []Inodos, inicio int64, n int, npos int) error {

	SB := SuperBloque{}
	BMInode := BitMapInodo{}

	BmInode_size := unsafe.Sizeof(BMInode)
	prueba := unsafe.Sizeof(SB)

	file, err := os.OpenFile(path, os.O_RDWR, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	offset := int64(inicio) + int64(prueba) + int64(n) + int64(BmInode_size+uintptr(n))
	if _, err := file.Seek(4100, 0); err != nil {
		return err
	}

	for i := 0; i < 2; i++ {
		offset1 := offset + int64(i)*int64(binary.Size(Inodos{}))
		if _, err := file.Seek(offset1, 0); err != nil {
			return err
		}
		if err := binary.Write(file, binary.LittleEndian, &inodo[i]); err != nil {
			return err
		}
	}

	return nil
}

func Escribir_BloqueCarpetas(path string, Carpeta []BloqueCarpetas, inicio int, n int) {

	SB := SuperBloque{}
	BMInode := BitMapInodo{}

	BmInode_size := unsafe.Sizeof(BMInode)
	prueba := unsafe.Sizeof(SB)

	//nuevoInicio := inicio + int(EBR_Size)

	offset := int64(inicio) + int64(prueba) + int64(n) + int64(BmInode_size+uintptr(n))
	offset1 := offset + int64(2)*int64(binary.Size(Inodos{}))

	file, err := os.OpenFile(path, os.O_RDWR, 0644)
	if err != nil {
		fmt.Println("¡¡ Error !! No se pudo acceder al disco")
		return
	}
	defer file.Close()

	if _, err := file.Seek(int64(offset1), 0); err != nil {
		fmt.Println(err)
		return
	}
	if err := binary.Write(file, binary.LittleEndian, &Carpeta); err != nil {
		fmt.Println(err)
		return
	}
}

func Escribir_BloqueArchivo(path string, Archivo BloqueArchivos, inicio int, n int) {

	SB := SuperBloque{}
	BMInode := BitMapInodo{}

	BmInode_size := unsafe.Sizeof(BMInode)
	prueba := unsafe.Sizeof(SB)

	otro := BloqueCarpetas{}

	abr := unsafe.Sizeof(otro)

	//nuevoInicio := inicio + int(EBR_Size)

	offset := int64(inicio) + int64(prueba) + int64(n) + int64(BmInode_size+uintptr(n))
	offset1 := offset + int64(2)*int64(binary.Size(Inodos{})) + int64(abr)

	file, err := os.OpenFile(path, os.O_RDWR, 0644)
	if err != nil {
		fmt.Println("¡¡ Error !! No se pudo acceder al disco")
		return
	}
	defer file.Close()

	if _, err := file.Seek(int64(offset1), 0); err != nil {
		fmt.Println(err)
		return
	}
	if err := binary.Write(file, binary.LittleEndian, &Archivo); err != nil {
		fmt.Println(err)
		return
	}
}

func Actualizar_SuperBloque(path string, SB SuperBloque, inicioP int) {

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

func verContenidoArchivo(path string, inicioP int) (string, string, error) {

	Tam_EBR := EBR{}
	EBR_Size := unsafe.Sizeof(Tam_EBR)

	file, err := os.OpenFile(path, os.O_RDONLY, 0644)
	if err != nil {
		fmt.Println("¡¡ Error !! No se pudo acceder al disco")
	}
	defer file.Close()

	nuevoinicio := inicioP + int(EBR_Size)

	if _, err := file.Seek(int64(nuevoinicio), 0); err != nil {
		fmt.Println(err)
	}

	var SB SuperBloque
	if err := binary.Read(file, binary.LittleEndian, &SB); err != nil {
		fmt.Println(err)
	}

	fsType := string(bytes.TrimRight(SB.S_inodes_count[:], string(0)))

	n, err := strconv.Atoi(fsType)

	BMInode := BitMapInodo{}

	BmInode_size := unsafe.Sizeof(BMInode)
	prueba := unsafe.Sizeof(SB)

	otro := BloqueCarpetas{}

	abr := unsafe.Sizeof(otro)

	offset := int64(inicioP) + int64(prueba) + int64(n) + int64(BmInode_size+uintptr(n))
	offset1 := offset + int64(2)*int64(binary.Size(Inodos{})) + int64(abr)

	if _, err := file.Seek(int64(offset1), 0); err != nil {
		fmt.Println(err)
		//return
	}

	var Archivo BloqueArchivos
	if err := binary.Read(file, binary.LittleEndian, &Archivo); err != nil {
		fmt.Println(err)
		//return
	}

	sakee := string(bytes.TrimRight(Archivo.B_content[:], string(0)))
	ggs := string(bytes.TrimRight(SB.S_inodes_count[:], string(0)))
	return sakee, ggs, nil

}

func verSB(path string, inicioP int) (string, error) {

	Tam_EBR := EBR{}
	EBR_Size := unsafe.Sizeof(Tam_EBR)

	file, err := os.OpenFile(path, os.O_RDONLY, 0644)
	if err != nil {
		fmt.Println("¡¡ Error !! No se pudo acceder al disco")
		//return
	}
	defer file.Close()

	nuevoinicio := inicioP + int(EBR_Size)

	if _, err := file.Seek(int64(nuevoinicio), 0); err != nil {
		fmt.Println(err)
		//return
	}

	var SB SuperBloque
	if err := binary.Read(file, binary.LittleEndian, &SB); err != nil {
		fmt.Println(err)
		//return
	}

	fsType := string(bytes.TrimRight(SB.S_inodes_count[:], string(0)))

	n, err := strconv.Atoi(fsType)

	BMInode := BitMapInodo{}

	BmInode_size := unsafe.Sizeof(BMInode)
	prueba := unsafe.Sizeof(SB)

	otro := BloqueCarpetas{}

	abr := unsafe.Sizeof(otro)

	offset := int64(inicioP) + int64(prueba) + int64(n) + int64(BmInode_size+uintptr(n))
	offset1 := offset + int64(2)*int64(binary.Size(Inodos{})) + int64(abr)

	if _, err := file.Seek(int64(offset1), 0); err != nil {
		fmt.Println(err)
		//return
	}

	var Archivo BloqueArchivos
	if err := binary.Read(file, binary.LittleEndian, &Archivo); err != nil {
		fmt.Println(err)
		//return
	}

	sakee := string(bytes.TrimRight(Archivo.B_content[:], string(0)))
	return sakee, nil

}

func verArchivo(path string, inicioP int) {
	Tam_EBR := EBR{}
	EBR_Size := unsafe.Sizeof(Tam_EBR)

	file, err := os.OpenFile(path, os.O_RDONLY, 0644)
	if err != nil {
		fmt.Println("¡¡ Error !! No se pudo acceder al disco")
		return
	}
	defer file.Close()

	nuevoinicio := inicioP + int(EBR_Size)

	if _, err := file.Seek(int64(nuevoinicio), 0); err != nil {
		fmt.Println(err)
		return
	}

	var SB SuperBloque
	if err := binary.Read(file, binary.LittleEndian, &SB); err != nil {
		fmt.Println(err)
		return
	}

	// fsType := string(bytes.TrimRight(SB.S_filesystem_type[:], string(0)))
	// fsType1 := string(bytes.TrimRight(SB.S_mtime[:], string(0)))
	// fsType2 := string(bytes.TrimRight(SB.S_magic[:], string(0)))
	// fmt.Println("File ", fsType)
	// fmt.Println("Time", fsType1)
	// fmt.Println("Magic", fsType2)

}

func comando_login(commandArray []string) string {
	encontrado := false
	respuesta := ""
	straux := ""
	id_buscar := ""
	user_buscar := ""
	pass_buscar := ""

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
		} else if strings.Contains(data, ">user=") {
			user_buscar = strings.Replace(data, ">user=", "", 1)
		} else if strings.Contains(data, ">pwd=") {
			pass_buscar = strings.Replace(data, ">pwd=", "", 1)
		}
	}

	if id_buscar == "" {
		fmt.Println("¡¡ Error !! No se ha especificado un id para el login")
		fmt.Println("")
		respuesta = "¡¡ Error !! No se ha especificado un id para el login"
		return respuesta
	}

	if user_buscar == "" {
		fmt.Println("¡¡ Error !! No se ha especificado un usuario para el login")
		fmt.Println("")
		respuesta = "¡¡ Error !! No se ha especificado un usuario para el login"
		return respuesta
	}

	if pass_buscar == "" {
		fmt.Println("¡¡ Error !! No se ha especificado una contraseña para el login")
		fmt.Println("")
		respuesta = "¡¡ Error !! No se ha especificado una contraseña para el login"
		return respuesta
	}

	// Verificar si la lista está vacía
	if len(usuarioLogeado) == 0 {
		existe, nodo := miLista.buscarPorID(id_buscar)
		if existe {

			// Se lee el contenido del primer archivo
			sakee, err := verSB(nodo.ruta, nodo.inicioparticion)
			if err != nil {
				// manejar error
			}

			lines := strings.Split(sakee, "\n")

			for _, line := range lines {
				if strings.Contains(line, user_buscar) {
					fields := strings.Split(line, ",")
					//fmt.Printf("Numero: %s, Tipo: %s, Usuario: %s, Contra: %s\n", fields[0], fields[1], fields[2], fields[3])
					fmt.Printf("Numero: %s, Tipo: %s, Grupo: %s\n", fields[0], fields[1], fields[2])
					if fields[1] == "U" {
						fmt.Printf("Es un usuario %s\n", fields[4])
						if fields[3] == user_buscar && fields[4] == pass_buscar {
							if fields[0] == "0" {
								fmt.Println("El usuario no existe")
								encontrado = false
								respuesta = "El usuario no existe"
								return respuesta
							} else {
								usuarioLogeado = append(usuarioLogeado, NodoLogin{id: id_buscar, nombre: user_buscar})
								encontrado = true
							}

						} else {
							fmt.Println("El usuario o contraseña no coincide ")
							encontrado = false
							respuesta = "El usuario o contraseña no coincide "
							return respuesta
						}
					}
				}
			}
			// fmt.Println("")
			// fmt.Println("*           Se ha iniciado sesion con el usuario: ", user_buscar, " 	*")
			// fmt.Println("")
			//

		} else {
			fmt.Println("No se encontró ningúna particion con ese ID")
			respuesta = "No se encontró ningúna particion con ese ID"
			return respuesta
		}

	} else {
		for _, usuario := range usuarioLogeado {
			fmt.Println("¡¡ Error !! El usuario ", usuario.nombre, " esta logeado, debes cerrar sesion primero")
			respuesta = "¡¡ Error !! El usuario " + usuario.nombre + " esta logeado, debes cerrar sesion primero"
		}
		return respuesta
	}

	if encontrado == false {
		respuesta = "¡¡ Error !! No se puedo iniciar sesion"
	} else {
		respuesta = "*           Se ha iniciado sesion con el usuario: " + user_buscar + " 	*"
		fmt.Println("")
		fmt.Println("*           Se ha iniciado sesion con el usuario: ", user_buscar, " 	*")
		fmt.Println("")
	}

	return respuesta

}

func nuevologin(id_buscar string, user_buscar string, pass_buscar string) string {
	encontrado := false
	respuesta := ""

	// Verificar si la lista está vacía
	if len(usuarioLogeado) == 0 {
		existe, nodo := miLista.buscarPorID(id_buscar)
		if existe {

			// Se lee el contenido del primer archivo
			sakee, err := verSB(nodo.ruta, nodo.inicioparticion)
			if err != nil {
				// manejar error
			}

			lines := strings.Split(sakee, "\n")

			for _, line := range lines {
				if strings.Contains(line, user_buscar) {
					fields := strings.Split(line, ",")
					//fmt.Printf("Numero: %s, Tipo: %s, Usuario: %s, Contra: %s\n", fields[0], fields[1], fields[2], fields[3])
					fmt.Printf("Numero: %s, Tipo: %s, Grupo: %s\n", fields[0], fields[1], fields[2])
					if fields[1] == "U" {
						fmt.Printf("Es un usuario %s\n", fields[4])
						if fields[3] == user_buscar && fields[4] == pass_buscar {
							if fields[0] != "0" {
								usuarioLogeado = append(usuarioLogeado, NodoLogin{id: id_buscar, nombre: user_buscar})
								encontrado = true
							} else {
								encontrado = false
							}

							//respuesta = "OK"
						} else {
							encontrado = false
							fmt.Println("El usuario o contraseña no coincide ")
							//respuesta = "El usuario o contraseña no coincide "
							//respuesta = "NO"
							return respuesta
						}
					}
				}
			}

		} else {
			fmt.Println("No se encontró ningúna particion con ese ID")
			//respuesta = "No se encontró ningúna particion con ese ID"
			respuesta = "NO"
			return respuesta
		}

	} else {
		for _, usuario := range usuarioLogeado {
			fmt.Println("¡¡ Error !! El usuario ", usuario.nombre, " esta logeado, debes cerrar sesion primero")
			respuesta = "¡¡ Error !! El usuario " + usuario.nombre + " esta logeado, debes cerrar sesion primero"
		}
		return respuesta
	}

	if encontrado == false {
		respuesta = "NO"
	} else {
		respuesta = "OK"
		fmt.Println("")
		fmt.Println("*           Se ha iniciado sesion con el usuario: ", user_buscar, " 	*")
		fmt.Println("")
	}

	return respuesta

}

func comando_logout() string {
	respuesta := ""
	if len(usuarioLogeado) == 0 {
		fmt.Println("¡¡ Error !! No hay ningun usuario con la sesion iniciada")
		respuesta = "¡¡ Error !! No hay ningun usuario con la sesion iniciada"
	} else {
		for _, usuario := range usuarioLogeado {
			fmt.Println("Sesion actual: ", usuario.nombre)
		}

		cerrar_sesion()
		fmt.Println("")
		fmt.Println("*                  Se ha cerrado la sesion                 *")
		fmt.Println("")
		respuesta = "*                  Se ha cerrado la sesion                 *"
	}

	return respuesta
}

func comando_mkgrp(commandArray []string) string {
	respuesta := ""
	straux := ""
	name_grupo := ""

	for i := 0; i < len(commandArray); i++ {
		data := commandArray[i]
		// if strings.HasPrefix(data, ">") {
		// 	// Convertir a minúsculas
		// 	data = strings.ToLower(data)
		// }

		if strings.Contains(data, ">name=") {
			straux = strings.Replace(data, ">name=", "", 1)
			name_grupo = straux

		}
	}

	if name_grupo == "" {
		fmt.Println("¡¡ Error !! No se ha especificado un nombre para el grupo")
		fmt.Println("")
		respuesta = "¡¡ Error !! No se ha especificado un nombre para el grupo"
		return respuesta
	}

	for _, usuario := range usuarioLogeado {
		if usuario.nombre == "root" {
			fmt.Println("Si puede crear un grupo")

			existe, nodo := miLista.buscarPorID(usuario.id)
			if existe {
				sakee, n, err := verContenidoArchivo(nodo.ruta, nodo.inicioparticion)
				if err != nil {
					// manejar error
				}

				caracteres_grupo := len(name_grupo)
				if caracteres_grupo > 10 {
					fmt.Println("¡¡ Error !! El nombre puede tener un maximo de 10 caracteres, el actual tiene : ", caracteres_grupo, " caracteres")
					respuesta = "¡¡ Error !! El nombre puede tener un maximo de 10 caracteres, el actual tiene : " + strconv.Itoa(caracteres_grupo) + " caracteres"
					return respuesta
				}

				int_n, err := strconv.Atoi(n)

				encontrado := false

				lineas := strings.Split(sakee, "\n")

				for i, linea := range lineas {
					// Si la línea contiene "U"
					if strings.Contains(linea, "G") {
						campos := strings.Split(linea, ",")
						if campos[2] == name_grupo {
							encontrado = true
							int_pp, err := strconv.Atoi(campos[0])
							if err != nil {
								// manejar error
							}
							if int_pp == 0 {

								numfinal := 0

								lines := strings.Split(sakee, "\n")
								nums := []int{}
								for _, line := range lines {
									fields := strings.Split(line, ",")
									if len(fields) >= 3 && fields[1] == "G" {
										num, err := strconv.Atoi(fields[0])
										if err == nil {
											nums = append(nums, num)
										}
									}
								}
								if len(nums) > 0 {
									max := findMax(nums)
									numfinal = max + 1
								} else {
									fmt.Println("No se encontraron números de grupo (G)")
								}
								if err != nil {
									fmt.Println("Error convirtiendo número:", err)
									return err.Error()
								}

								// Actualizar el número al siguiente
								campos[0] = strconv.Itoa(numfinal)

								// Unir los campos por coma
								lineas[i] = strings.Join(campos, ",")

								resultado := strings.Join(lineas, "\n")

								fmt.Println(resultado)

								Archivo := BloqueArchivos{}
								contenidoarchivo := resultado
								copy(Archivo.B_content[:], []byte(contenidoarchivo))

								Escribir_BloqueArchivo(nodo.ruta, Archivo, nodo.inicioparticion, int_n)

								fmt.Println("")
								fmt.Println("*           Se ha agregar el grupo ", name_grupo, " del archivo        *")
								fmt.Println("")

								respuesta = "*           Se ha agregado el grupo " + name_grupo + " del archivo        *"
								//respuesta = "Creado"
								return respuesta
							}

						}
					}
				}
				if encontrado {
					fmt.Println("¡¡ Error !! Ya existe ese grupo")
					respuesta = "¡¡ Error !! Ya existe ese grupo"
					return respuesta
				} else {
					numfinal := 0

					lines := strings.Split(sakee, "\n")
					nums := []int{}
					for _, line := range lines {
						fields := strings.Split(line, ",")
						if len(fields) >= 3 && fields[1] == "G" {
							num, err := strconv.Atoi(fields[0])
							if err == nil {
								nums = append(nums, num)
							}
						}
					}
					if len(nums) > 0 {
						max := findMax(nums)
						numfinal = max + 1
					} else {
						fmt.Println("No se encontraron números de grupo (G)")
					}
					pruebas := strconv.Itoa(numfinal) + ",G," + name_grupo + "\n"

					concat := sakee + pruebas
					Archivo := BloqueArchivos{}
					contenidoarchivo := concat
					copy(Archivo.B_content[:], []byte(contenidoarchivo))

					Escribir_BloqueArchivo(nodo.ruta, Archivo, nodo.inicioparticion, int_n)

					fmt.Println("")
					fmt.Println("*           Se ha agregado el grupo ", name_grupo, " al archivo        *")
					fmt.Println("")
					respuesta = "*           Se ha agregado el grupo " + name_grupo + " al archivo        *"

				}

			}

		} else {
			fmt.Println("¡¡ Error !! Este comando solo lo puede ejecutar el usuario root")
			respuesta = "¡¡ Error !! Este comando solo lo puede ejecutar el usuario root"
		}
	}

	return respuesta

}

func comando_rmgrp(commandArray []string) string {
	respuesta := ""
	straux := ""
	name_grupo := ""

	for i := 0; i < len(commandArray); i++ {
		data := commandArray[i]
		// if strings.HasPrefix(data, ">") {
		// 	// Convertir a minúsculas
		// 	data = strings.ToLower(data)
		// }

		if strings.Contains(data, ">name=") {
			straux = strings.Replace(data, ">name=", "", 1)
			name_grupo = straux

		}
	}

	if name_grupo == "" {
		fmt.Println("¡¡ Error !! No se ha especificado un nombre para el grupo a eliminar")
		fmt.Println("")
		respuesta = "¡¡ Error !! No se ha especificado un nombre para el grupo a eliminar"
		return respuesta
	}

	for _, usuario := range usuarioLogeado {
		if usuario.nombre == "root" {

			existe, nodo := miLista.buscarPorID(usuario.id)
			if existe {
				sakee, n, err := verContenidoArchivo(nodo.ruta, nodo.inicioparticion)
				if err != nil {
					// manejar error
				}

				int_n, err := strconv.Atoi(n)

				lineas := strings.Split(sakee, "\n")

				encontrado := false

				for i, linea := range lineas {
					// Si la línea contiene "U"
					if strings.Contains(linea, "G") {
						campos := strings.Split(linea, ",")

						if name_grupo == "root" || name_grupo == "ROOT" {
							fmt.Println("¡¡ Error !! No se puede eliminar el grupo root")
							respuesta = "¡¡ Error !! No se puede eliminar el grupo root"
							return respuesta
						}
						if campos[2] == name_grupo {
							encontrado = true
							int_pp, err := strconv.Atoi(campos[0])
							if int_pp == 0 {
								fmt.Println("¡¡ Error !! No existe ese nombre de grupo")
								respuesta = "¡¡ Error !! No existe ese nombre de grupo"
								return respuesta
							}
							numero, err := strconv.Atoi(campos[0])
							if err != nil {
								fmt.Println("Error convirtiendo número:", err)
								return err.Error()
							}

							fmt.Println("Que sale? ", numero)

							// Actualizar el número a 0
							campos[0] = "0"

							// Unir los campos por coma
							lineas[i] = strings.Join(campos, ",")
						}
					}
				}

				if encontrado == false {
					fmt.Println("¡¡ Error !! No se encuentra un grupo con ese nombre")
					respuesta = "¡¡ Error !! No se encuentra un grupo con ese nombre"
					return respuesta
				}

				// Unir las líneas por "\n"
				resultado := strings.Join(lineas, "\n")

				fmt.Println(resultado)

				Archivo := BloqueArchivos{}
				contenidoarchivo := resultado
				copy(Archivo.B_content[:], []byte(contenidoarchivo))

				Escribir_BloqueArchivo(nodo.ruta, Archivo, nodo.inicioparticion, int_n)

				fmt.Println("")
				fmt.Println("*           Se ha eliminado el grupo ", name_grupo, " del archivo        *")
				fmt.Println("")

				respuesta = "*           Se ha eliminado el grupo " + name_grupo + " del archivo        *"

			}
		} else {
			fmt.Println("¡¡ Error !! Este comando solo lo puede ejecutar el usuario root")
			respuesta = "¡¡ Error !! Este comando solo lo puede ejecutar el usuario root"
		}
	}

	return respuesta
}

func comando_mkusr(commandArray []string) string {
	user_encontrado := false
	respuesta := ""
	straux := ""
	name_user := ""
	pass_user := ""
	name_grupo := ""

	for i := 0; i < len(commandArray); i++ {
		data := commandArray[i]
		// if strings.HasPrefix(data, ">") {
		// 	// Convertir a minúsculas
		// 	data = strings.ToLower(data)
		// }

		if strings.Contains(data, ">user=") {
			straux = strings.Replace(data, ">user=", "", 1)
			name_user = straux

		} else if strings.Contains(data, ">pwd=") {
			straux = strings.Replace(data, ">pwd=", "", 1)
			pass_user = straux

		} else if strings.Contains(data, ">grp=") {
			straux = strings.Replace(data, ">grp=", "", 1)
			name_grupo = straux

		}
	}

	if name_user == "" {
		fmt.Println("¡¡ Error !! No se ha especificado un nombre para el usuario")
		fmt.Println("")
		respuesta = "¡¡ Error !! No se ha especificado un nombre para el usuario"
		return respuesta
	}

	if pass_user == "" {
		fmt.Println("¡¡ Error !! No se ha especificado una contraseña para el usuario")
		fmt.Println("")
		respuesta = "¡¡ Error !! No se ha especificado una contraseña para el usuario"
		return respuesta
	}

	if name_grupo == "" {
		fmt.Println("¡¡ Error !! No se ha especificado un nombre para el grupo")
		fmt.Println("")
		respuesta = "¡¡ Error !! No se ha especificado un nombre para el grupo"
		return respuesta
	}

	for _, usuario := range usuarioLogeado {
		if usuario.nombre == "root" {
			fmt.Println("Si puede crear un usuario")

			existe, nodo := miLista.buscarPorID(usuario.id)
			if existe {
				sakee, n, err := verContenidoArchivo(nodo.ruta, nodo.inicioparticion)
				if err != nil {
					// manejar error
				}

				caracteres_usuario := len(name_user)
				caracteres_contra := len(pass_user)
				caracteres_grupo := len(name_grupo)
				if caracteres_usuario > 10 {
					fmt.Println("¡¡ Error !! El usuario puede tener un maximo de 10 caracteres, el actual tiene : ", caracteres_usuario, " caracteres")
					respuesta = "¡¡ Error !! El usuario puede tener un maximo de 10 caracteres, el actual tiene : " + strconv.Itoa(caracteres_usuario) + " caracteres"
					return respuesta
				}

				if caracteres_contra > 10 {
					fmt.Println("¡¡ Error !! La contraseña puede tener un maximo de 10 caracteres, la actual tiene : ", caracteres_usuario, " caracteres")
					respuesta = "¡¡ Error !! La contraseña puede tener un maximo de 10 caracteres, la actual tiene : " + strconv.Itoa(caracteres_usuario) + " caracteres"
					return respuesta
				}

				if caracteres_grupo > 10 {
					fmt.Println("¡¡ Error !! El grupo puede tener un maximo de 10 caracteres, el actual tiene : ", caracteres_usuario, " caracteres")
					respuesta = "¡¡ Error !! El grupo puede tener un maximo de 10 caracteres, el actual tiene : " + strconv.Itoa(caracteres_usuario) + " caracteres"
					return respuesta
				}

				int_n, err := strconv.Atoi(n)

				fmt.Println(int_n)

				lines := strings.Split(sakee, "\n")

				encontrado := false

				for _, line := range lines {
					if strings.Contains(line, name_user) {
						fields := strings.Split(line, ",")
						if fields[1] == "U" {
							if fields[0] != "0" {
								fmt.Println("¡¡ Ya existe !!")
								respuesta = "¡¡ Ya existe !!"
								return respuesta
							}
						}
					}
				}

				for _, line := range lines {
					if strings.Contains(line, name_grupo) {
						fields := strings.Split(line, ",")
						// //fmt.Printf("Numero: %s, Tipo: %s, Usuario: %s, Contra: %s\n", fields[0], fields[1], fields[2], fields[3])
						// //fmt.Printf("Numero: %s, Tipo: %s, Usuario: %s\n", fields[0], fields[1], fields[2])
						if fields[1] == "G" {
							encontrado = true
						}
					}
				}

				if encontrado == false {
					fmt.Println("¡¡ Error !! No se encuentra el grupo al que se quiere agregar el usuario")
					respuesta = "¡¡ Error !! No se encuentra el grupo al que se quiere agregar el usuario"
					return respuesta
				} else {

					int_n, err := strconv.Atoi(n)
					if err != nil {
						// manejar error
					}

					lineas := strings.Split(sakee, "\n")

					for i, linea := range lineas {
						// Si la línea contiene "U"
						if strings.Contains(linea, "U") {
							campos := strings.Split(linea, ",")
							if campos[4] == name_grupo {
								user_encontrado = true
								int_pp, err := strconv.Atoi(campos[0])
								if err != nil {
									// manejar error
								}
								if int_pp == 0 {

									numfinal := 0

									lines := strings.Split(sakee, "\n")
									nums := []int{}
									for _, line := range lines {
										fields := strings.Split(line, ",")
										if len(fields) >= 5 && fields[1] == "U" {
											num, err := strconv.Atoi(fields[0])
											if err == nil {
												nums = append(nums, num)
											}
										}
									}
									if len(nums) > 0 {
										max := findMax(nums)
										numfinal = max + 1
									} else {
										fmt.Println("No se encontraron números de grupo (U)")
									}
									if err != nil {
										fmt.Println("Error convirtiendo número:", err)
										return err.Error()
									}

									// Actualizar el número al siguiente
									campos[0] = strconv.Itoa(numfinal)

									// Unir los campos por coma
									lineas[i] = strings.Join(campos, ",")

									resultado := strings.Join(lineas, "\n")

									fmt.Println(resultado)

									Archivo := BloqueArchivos{}
									contenidoarchivo := resultado
									copy(Archivo.B_content[:], []byte(contenidoarchivo))

									Escribir_BloqueArchivo(nodo.ruta, Archivo, nodo.inicioparticion, int_n)

									fmt.Println("")
									fmt.Println("*           Se ha agregado el usuario ", name_user, " al archivo        *")
									fmt.Println("")
									respuesta = "*           Se ha agregado el usuario " + name_user + " al archivo        *"
									//respuesta = "Creado"
									return respuesta
								} else {
									fmt.Println("¡¡ Error !! Ya existe ese usuario")
									respuesta = "¡¡ Error !! Ya existe ese usuario"
									return respuesta
								}
							}
						}
					}
				}

				if user_encontrado {
					fmt.Println("¡¡ Error !! Usuario ya existe")
					respuesta = "¡¡ Error !! Usuario ya existe"
					return respuesta
				} else {
					numfinal := 0

					lines := strings.Split(sakee, "\n")
					nums := []int{}
					for _, line := range lines {
						fields := strings.Split(line, ",")
						if len(fields) >= 3 && fields[1] == "U" {
							num, err := strconv.Atoi(fields[0])
							if err == nil {
								nums = append(nums, num)
							}
						}
					}
					if len(nums) > 0 {
						max := findMax(nums)
						numfinal = max + 1
					} else {
						fmt.Println("No se encontraron números de grupo (U)")
					}
					pruebas := strconv.Itoa(numfinal) + ",U," + name_grupo + "," + name_user + "," + pass_user + "\n"

					concat := sakee + pruebas
					//fmt.Println("Sale ", concat)
					Archivo := BloqueArchivos{}
					contenidoarchivo := concat
					copy(Archivo.B_content[:], []byte(contenidoarchivo))

					Escribir_BloqueArchivo(nodo.ruta, Archivo, nodo.inicioparticion, int_n)

					fmt.Println("")
					fmt.Println("*           Se ha agregado el usuario ", name_user, " al archivo        *")
					fmt.Println("")
					respuesta = "*           Se ha agregado el usuario " + name_user + " al archivo        *"
				}
			}
		} else {
			fmt.Println("¡¡ Error !! Este comando solo lo puede ejecutar el usuario root")
			respuesta = "¡¡ Error !! Este comando solo lo puede ejecutar el usuario root"
		}
	}

	return respuesta

}

func comando_rmusr(commandArray []string) string {
	respuesta := ""
	straux := ""
	name_user := ""

	for i := 0; i < len(commandArray); i++ {
		data := commandArray[i]
		// if strings.HasPrefix(data, ">") {
		// 	// Convertir a minúsculas
		// 	data = strings.ToLower(data)
		// }

		if strings.Contains(data, ">user=") {
			straux = strings.Replace(data, ">user=", "", 1)
			name_user = straux

		}
	}

	if name_user == "" {
		fmt.Println("¡¡ Error !! No se ha especificado un nombre para el usuario a eliminar")
		fmt.Println("")
		respuesta = "¡¡ Error !! No se ha especificado un nombre para el grupo a eliminar"
		return respuesta
	}

	for _, usuario := range usuarioLogeado {
		if usuario.nombre == "root" {

			existe, nodo := miLista.buscarPorID(usuario.id)
			if existe {
				sakee, n, err := verContenidoArchivo(nodo.ruta, nodo.inicioparticion)
				if err != nil {
					// manejar error
				}

				int_n, err := strconv.Atoi(n)

				lineas := strings.Split(sakee, "\n")

				encontrado := false

				for i, linea := range lineas {
					// Si la línea contiene "U"
					if strings.Contains(linea, "U") {
						campos := strings.Split(linea, ",")

						if name_user == "root" || name_user == "ROOT" {
							fmt.Println("¡¡ Error !! No se puede eliminar el usuario root")
							respuesta = "¡¡ Error !! No se puede eliminar el usuario root"
							return respuesta
						}
						if campos[3] == name_user {
							encontrado = true
							int_pp, err := strconv.Atoi(campos[0])
							if int_pp == 0 {
								fmt.Println("¡¡ Error !! No existe ese nombre de usuario")
								respuesta = "¡¡ Error !! No existe ese nombre de usuario"
								return respuesta
							}
							numero, err := strconv.Atoi(campos[0])
							if err != nil {
								fmt.Println("Error convirtiendo número:", err)
								return err.Error()
							}

							fmt.Println("Que sale? ", numero)

							// Actualizar el número a 0
							campos[0] = "0"

							// Unir los campos por coma
							lineas[i] = strings.Join(campos, ",")
						}
					}
				}

				if encontrado == false {
					fmt.Println("¡¡ Error !! No se encuentra un usuario con ese nombre")
					respuesta = "¡¡ Error !! No se encuentra un usuario con ese nombre"
					return respuesta
				}

				// Unir las líneas por "\n"
				resultado := strings.Join(lineas, "\n")

				fmt.Println(resultado)

				Archivo := BloqueArchivos{}
				contenidoarchivo := resultado
				copy(Archivo.B_content[:], []byte(contenidoarchivo))

				Escribir_BloqueArchivo(nodo.ruta, Archivo, nodo.inicioparticion, int_n)

				fmt.Println("")
				fmt.Println("*           Se ha eliminado el usuario ", name_user, " del archivo        *")
				fmt.Println("")
				respuesta = "*           Se ha eliminado el usuario " + name_user + " del archivo        *"

			}
		} else {
			fmt.Println("¡¡ Error !! Este comando solo lo puede ejecutar el usuario root")
			respuesta = "¡¡ Error !! Este comando solo lo puede ejecutar el usuario root"
		}
	}

	return respuesta
}

func comando_rep(commandArray []string) string {
	respuesta := ""
	straux := ""
	name_rep := ""
	rutaa := ""
	id_buscar := ""
	ruta_v := ""

	for i := 0; i < len(commandArray); i++ {
		data := commandArray[i]
		// if strings.HasPrefix(data, ">") {
		// 	// Convertir a minúsculas
		// 	data = strings.ToLower(data)
		// }

		if strings.Contains(data, ">name=") {
			straux = strings.Replace(data, ">name=", "", 1)
			name_rep = straux

		} else if strings.Contains(data, ">path=") {
			straux = strings.Replace(data, ">path=", "", 1)
			rutaa = straux

		} else if strings.Contains(data, ">id=") {
			straux = strings.Replace(data, ">id=", "", 1)
			id_buscar = straux

		} else if strings.Contains(data, ">ruta=") {
			straux = strings.Replace(data, ">ruta=", "", 1)
			ruta_v = straux

		}
	}

	if name_rep == "" {
		fmt.Println("¡¡ Error !! No se ha especificado un tipo de reporte ")
		fmt.Println("")
		respuesta = "¡¡ Error !! No se ha especificado un tipo de reporte "
		return respuesta
	}

	if rutaa == "" {
		fmt.Println("¡¡ Error !! No se ha especificado una ruta para guardar el reporte ")
		fmt.Println("")
		respuesta = "¡¡ Error !! No se ha especificado una ruta para guardar el reporte "
		return respuesta
	}

	if id_buscar == "" {
		fmt.Println("¡¡ Error !! No se ha especificado un id para generar los datos del reporte ")
		fmt.Println("")
		respuesta = "¡¡ Error !! No se ha especificado un id para generar los datos del reporte "
		return respuesta
	}

	if ruta_v == "" {
		//Es opcional este parametro
	}

	if name_rep == "disk" || name_rep == "DISK" {
		fmt.Println("Entra al reporte disk")
		existe, nodo := miLista.buscarPorID(id_buscar)
		if existe {
			respuesta = reporte_disk(nodo, rutaa)

		} else {
			fmt.Println("¡¡ Error !! No se encontró ningúna particion con ese ID")
			respuesta = "¡¡ Error !! No se encontró ningúna particion con ese ID"
			return respuesta
		}
	} else if name_rep == "tree" || name_rep == "TREE" {
		fmt.Println("Entra al reporte tree")
		existe, nodo := miLista.buscarPorID(id_buscar)
		if existe {
			respuesta = reporte_tree(nodo, rutaa)

		} else {
			fmt.Println("¡¡ Error !! No se encontró ningúna particion con ese ID")
			respuesta = "¡¡ Error !! No se encontró ningúna particion con ese ID"
			return respuesta
		}
	} else if name_rep == "file" || name_rep == "FILE" {
		fmt.Println("Entra al reporte file")
	} else if name_rep == "sb" || name_rep == "SB" {
		existe, nodo := miLista.buscarPorID(id_buscar)
		if existe {
			respuesta = reporte_sb(nodo, rutaa)

		} else {
			fmt.Println("¡¡ Error !! No se encontró ningúna particion con ese ID")
			respuesta = "¡¡ Error !! No se encontró ningúna particion con ese ID"
			return respuesta
		}
	} else {
		fmt.Println("¡¡ Error !! el nombre de reporte no existe en este proyecto")
		respuesta = "¡¡ Error !! el nombre de reporte no existe en este proyecto"
		return respuesta
	}

	return respuesta

}

func reporte_disk(nodoActual *NodoMount, rutaa string) string {
	respuesta := ""

	file, err := os.OpenFile(nodoActual.ruta, os.O_RDONLY, 0644)
	if err != nil {
		fmt.Println("¡¡ Error !! No se pudo acceder al disco")
	}
	defer file.Close()

	if _, err := file.Seek(int64(0), 0); err != nil {
		fmt.Println(err)
	}

	var PurbeaD MBR
	if err := binary.Read(file, binary.LittleEndian, &PurbeaD); err != nil {
		fmt.Println(err)
	}
	string_tamano := string(bytes.TrimRight(PurbeaD.Mbr_tamano[:], string(0)))
	fmt.Println("Tam : ", string_tamano)

	trimmed_string_tamano := strings.TrimRightFunc(string_tamano, func(r rune) bool { return r == '\x00' })
	tamano, err := strconv.Atoi(trimmed_string_tamano)
	if err != nil {
		fmt.Println("Error:", err)
	}

	fmt.Println("Tamm ? ", tamano)

	nombreedisco := filepath.Base(nodoActual.ruta)
	nombre := filepath.Base(rutaa)                                   // obtiene el nombre del archivo con la extensión
	nombreSinExt := strings.TrimSuffix(nombre, filepath.Ext(nombre)) // remueve la extensión

	// Creacion, escritura y cierre de archivo
	directorio := rutaa[:strings.LastIndex(rutaa, "/")+1]

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
	fmt.Println(nombreSinExt)

	discop := PurbeaD

	particion := [4]Partition{
		discop.Mbr_partition_1,
		discop.Mbr_partition_2,
		discop.Mbr_partition_3,
		discop.Mbr_partition_4,
	}

	fmt.Println("A ", string(particion[0].Part_name[:]))
	fmt.Println("B ", string(particion[1].Part_name[:]))
	fmt.Println("C ", string(particion[2].Part_name[:]))
	fmt.Println("D ", string(particion[3].Part_name[:]))

	tipo_p1 := string(particion[0].Part_type[:])
	tipo_p2 := string(particion[1].Part_type[:])
	tipo_p3 := string(particion[2].Part_type[:])
	tipo_p4 := string(particion[3].Part_type[:])

	ajuste_p1 := string(particion[0].Part_fit[:])

	dot := "digraph G {\n"
	dot = dot + "labelloc=\"t\"\n"
	dot = dot + "label=\"" + nombreedisco + "\"\n"
	dot = dot + "parent [\n"
	dot = dot + "shape=plaintext\n"
	dot = dot + "label=<\n"
	dot = dot + "<table border=\"1\" cellborder=\"1\">\n"
	dot = dot + "<tr> <td rowspan='3'>MBR</td>\n"

	existeExtendida1 := false
	existeExtendida2 := false
	existeExtendida3 := false
	existeExtendida4 := false

	string_sizeP1 := string(bytes.TrimRight(particion[0].Part_size[:], string(0)))
	tamano_P1, err := strconv.Atoi(string_sizeP1)
	string_sizeP2 := string(bytes.TrimRight(particion[1].Part_size[:], string(0)))
	tamano_P2, err := strconv.Atoi(string_sizeP2)
	string_sizeP3 := string(bytes.TrimRight(particion[2].Part_size[:], string(0)))
	tamano_P3, err := strconv.Atoi(string_sizeP3)
	string_sizeP4 := string(bytes.TrimRight(particion[3].Part_size[:], string(0)))
	tamano_P4, err := strconv.Atoi(string_sizeP4)

	salida := 0
	salida2 := 0
	salida3 := 0
	salida4 := 0
	if err != nil {
		fmt.Println("Error:", err)
	}

	if ajuste_p1 != "" {
		if tipo_p1 == "E" {
			salida = (tamano_P1 * 100) / tamano
			dot = dot + "<td colspan=\"6\" rowspan=\"1\">Extendida</td>"
			existeExtendida1 = true
		} else {
			salida = (tamano_P1 * 100) / tamano
			porcentaje1 := strconv.Itoa(salida)
			dot = dot + "<td rowspan=\"3\">Primaria <br/>" + porcentaje1 + "%" + " del disco</td>"
		}

		if tipo_p2 == "" {
			resta := 100 - salida
			espaciolibre := strconv.Itoa(resta)
			dot = dot + "<td rowspan=\"2\" port='libre'>Espacio Libre <br/>" + espaciolibre + "%" + "del disco</td>\n"

		} else if tipo_p2 == "E" {
			salida2 = (tamano_P2 * 100) / tamano
			dot = dot + "<td colspan=\"6\" rowspan=\"1\">Extendida</td>"
			existeExtendida2 = true
		} else {
			salida2 = (tamano_P2 * 100) / tamano
			porcentaje2 := strconv.Itoa(salida2)
			dot = dot + "<td rowspan=\"3\">Primaria <br/>" + porcentaje2 + "%" + " del disco</td>"
		}

		if tipo_p3 == "" {
			resta := 100 - (salida + salida2)
			espaciolibre := strconv.Itoa(resta)
			dot = dot + "<td rowspan=\"2\" port='libre'>Espacio Libre <br/>" + espaciolibre + "%" + "del disco</td>\n"
		} else if tipo_p3 == "E" {
			salida3 = (tamano_P3 * 100) / tamano
			dot = dot + "<td colspan=\"6\" rowspan=\"1\">Extendida</td>"
			existeExtendida3 = true
		} else {
			salida3 = (tamano_P3 * 100) / tamano
			porcentaje3 := strconv.Itoa(salida3)
			dot = dot + "<td rowspan=\"3\">Primaria <br/>" + porcentaje3 + "%" + " del disco</td>"
		}

		if tipo_p4 == "" {
			resta := 100 - (salida + salida2 + salida3)
			espaciolibre := strconv.Itoa(resta)
			dot = dot + "<td rowspan=\"2\" port='libre'>Espacio Libre <br/>" + espaciolibre + "%" + "del disco</td>\n"

		} else if tipo_p4 == "E" {
			salida4 = (tamano_P4 * 100) / tamano
			dot = dot + "<td colspan=\"6\" rowspan=\"1\">Extendida</td>"
			resta := 100 - (salida + salida2 + salida3 + salida4)
			espaciolibre := strconv.Itoa(resta)
			if resta > 0 {
				dot = dot + "<td rowspan=\"2\" port='libre'>Espacio Libre <br/>" + espaciolibre + "%" + "del disco</td>\n"
			}
			existeExtendida4 = true
		} else {
			salida4 = (tamano_P4 * 100) / tamano
			porcentaje4 := strconv.Itoa(salida4)
			dot = dot + "<td rowspan=\"3\">Primaria <br/>" + porcentaje4 + "%" + " del disco</td>"
			resta := 100 - (salida + salida2 + salida3 + salida4)
			espaciolibre := strconv.Itoa(resta)
			if resta > 0 {
				dot = dot + "<td rowspan=\"2\" port='libre'>Espacio Libre <br/>" + espaciolibre + "%" + "del disco</td>\n"
			}

		}

		if existeExtendida1 == true {

			file, err := os.OpenFile(nodoActual.ruta, os.O_RDONLY, 0644)
			if err != nil {
				fmt.Println("¡¡ Error !! No se pudo acceder al disco")
				respuesta = "¡¡ Error !! No se pudo acceder al disco"
				return respuesta
			}
			defer file.Close()

			if _, err := file.Seek(int64(nodoActual.inicioparticion), 0); err != nil {
				fmt.Println(err)
				return err.Error()
			}

			var PruebaEBR EBR
			if err := binary.Read(file, binary.LittleEndian, &PruebaEBR); err != nil {
				fmt.Println(err)
				return err.Error()
			}

			fsType2 := string(bytes.TrimRight(PruebaEBR.Part_name[:], string(0)))
			fmt.Println(fsType2)
			siguienteE := string(bytes.TrimRight(PruebaEBR.Part_next[:], string(0)))
			INT_P, err := strconv.Atoi(siguienteE)

			string_sizeP1 := string(bytes.TrimRight(particion[0].Part_size[:], string(0)))
			tamano_P1, err := strconv.Atoi(string_sizeP1)
			if err != nil {
				fmt.Println("Error:", err)
			}
			porcentajeP1 := (tamano_P1 * 100) / tamano
			if INT_P == -1 {
				dot = dot + "</tr>\n"
				dot = dot + "<tr>\n"
				nombrelog := string(bytes.TrimRight(PruebaEBR.Part_name[:], string(0)))
				size_p1 := string(bytes.TrimRight(PruebaEBR.Part_size[:], string(0)))
				int_size_p1, err := strconv.Atoi(size_p1)
				if err != nil {
					fmt.Println("Error:", err)
				}

				salidaP := (int_size_p1 * 100) / tamano
				porcentaje1 := strconv.Itoa(salidaP)

				espaciolibre := porcentajeP1 - salidaP
				str_espaciolinre := strconv.Itoa(espaciolibre)

				dot = dot + "<td rowspan=\"2\" port='log1'>EBR</td><td rowspan=\"2\" port='log2'>" + nombrelog + "<br/>" + porcentaje1 + "%" + "del disco</td>\n"
				dot = dot + "<td rowspan=\"2\" port='libre'>Espacio Libre <br/>" + str_espaciolinre + "%" + "del disco</td>\n"
			} else {
				dot = dot + "</tr>\n"
				dot = dot + "<tr>\n"
				nombrelog := string(bytes.TrimRight(PruebaEBR.Part_name[:], string(0)))
				size_p1 := string(bytes.TrimRight(PruebaEBR.Part_size[:], string(0)))
				int_size_p1, err := strconv.Atoi(size_p1)
				if err != nil {
					fmt.Println("Error:", err)
				}

				salidaP := (int_size_p1 * 100) / tamano
				porcentaje1 := strconv.Itoa(salidaP)

				if _, err := file.Seek(int64(nodoActual.inicioparticion+30), 0); err != nil {
					fmt.Println(err)
					return err.Error()
				}

				var SiguienteEBR EBR
				if err := binary.Read(file, binary.LittleEndian, &SiguienteEBR); err != nil {
					fmt.Println(err)
					return err.Error()
				}

				nombrelog2 := string(bytes.TrimRight(SiguienteEBR.Part_name[:], string(0)))
				size_p2 := string(bytes.TrimRight(SiguienteEBR.Part_size[:], string(0)))
				int_size_p2, err := strconv.Atoi(size_p2)
				if err != nil {
					fmt.Println("Error:", err)
				}

				salidaP2 := (int_size_p2 * 100) / tamano
				porcentaje2 := strconv.Itoa(salidaP2)

				espaciolibre := porcentajeP1 - (salidaP + salidaP2)
				str_espaciolinre := strconv.Itoa(espaciolibre)

				dot = dot + "<td rowspan=\"2\" port='log1'>EBR</td><td rowspan=\"2\" port='log'>" + nombrelog + "<br/>" + porcentaje1 + "%" + "del disco</td>\n"

				dot = dot + "<td rowspan=\"2\" port='log1'>EBR</td><td rowspan=\"2\" port='log2'>" + nombrelog2 + "<br/>" + porcentaje2 + "%" + "del disco</td>\n"

				dot = dot + "<td rowspan=\"2\" port='libre'>Espacio Libre <br/>" + str_espaciolinre + "%" + "del disco</td>\n"
			}

		} else if existeExtendida2 == true {

			file, err := os.OpenFile(nodoActual.ruta, os.O_RDONLY, 0644)
			if err != nil {
				fmt.Println("¡¡ Error !! No se pudo acceder al disco")
				respuesta = "¡¡ Error !! No se pudo acceder al disco"
				return respuesta
			}
			defer file.Close()

			if _, err := file.Seek(int64(nodoActual.inicioparticion), 0); err != nil {
				fmt.Println(err)
				return err.Error()
			}

			var PruebaEBR EBR
			if err := binary.Read(file, binary.LittleEndian, &PruebaEBR); err != nil {
				fmt.Println(err)
				return err.Error()
			}

			fsType2 := string(bytes.TrimRight(PruebaEBR.Part_name[:], string(0)))
			fmt.Println(fsType2)
			siguienteE := string(bytes.TrimRight(PruebaEBR.Part_next[:], string(0)))
			INT_P, err := strconv.Atoi(siguienteE)

			string_sizeP1 := string(bytes.TrimRight(particion[1].Part_size[:], string(0)))
			tamano_P2, err := strconv.Atoi(string_sizeP1)
			if err != nil {
				fmt.Println("Error:", err)
			}
			porcentajeP1 := (tamano_P2 * 100) / tamano
			if INT_P == -1 {
				dot = dot + "</tr>\n"
				dot = dot + "<tr>\n"
				nombrelog := string(bytes.TrimRight(PruebaEBR.Part_name[:], string(0)))
				size_p1 := string(bytes.TrimRight(PruebaEBR.Part_size[:], string(0)))
				int_size_p1, err := strconv.Atoi(size_p1)
				if err != nil {
					fmt.Println("Error:", err)
				}

				salidaP := (int_size_p1 * 100) / tamano
				porcentaje2 := strconv.Itoa(salidaP)

				espaciolibre := porcentajeP1 - salidaP
				str_espaciolinre := strconv.Itoa(espaciolibre)

				dot = dot + "<td rowspan=\"2\" port='log1'>EBR</td><td rowspan=\"2\" port='log2'>" + nombrelog + "<br/>" + porcentaje2 + "%" + "del disco</td>\n"
				dot = dot + "<td rowspan=\"2\" port='libre'>Espacio Libre <br/>" + str_espaciolinre + "%" + "del disco</td>\n"
			} else {
				dot = dot + "</tr>\n"
				dot = dot + "<tr>\n"
				nombrelog := string(bytes.TrimRight(PruebaEBR.Part_name[:], string(0)))
				size_p1 := string(bytes.TrimRight(PruebaEBR.Part_size[:], string(0)))
				int_size_p1, err := strconv.Atoi(size_p1)
				if err != nil {
					fmt.Println("Error:", err)
				}

				salidaP := (int_size_p1 * 100) / tamano
				porcentaje1 := strconv.Itoa(salidaP)

				if _, err := file.Seek(int64(nodoActual.inicioparticion+30), 0); err != nil {
					fmt.Println(err)
					return err.Error()
				}

				var SiguienteEBR EBR
				if err := binary.Read(file, binary.LittleEndian, &SiguienteEBR); err != nil {
					fmt.Println(err)
					return err.Error()
				}

				nombrelog2 := string(bytes.TrimRight(SiguienteEBR.Part_name[:], string(0)))
				size_p2 := string(bytes.TrimRight(SiguienteEBR.Part_size[:], string(0)))
				int_size_p2, err := strconv.Atoi(size_p2)
				if err != nil {
					fmt.Println("Error:", err)
				}

				salidaP2 := (int_size_p2 * 100) / tamano
				porcentaje2 := strconv.Itoa(salidaP2)

				espaciolibre := porcentajeP1 - (salidaP + salidaP2)
				str_espaciolinre := strconv.Itoa(espaciolibre)

				dot = dot + "<td rowspan=\"2\" port='log1'>EBR</td><td rowspan=\"2\" port='log'>" + nombrelog + "<br/>" + porcentaje1 + "%" + "del disco</td>\n"

				dot = dot + "<td rowspan=\"2\" port='log1'>EBR</td><td rowspan=\"2\" port='log2'>" + nombrelog2 + "<br/>" + porcentaje2 + "%" + "del disco</td>\n"

				dot = dot + "<td rowspan=\"2\" port='libre'>Espacio Libre <br/>" + str_espaciolinre + "%" + "del disco</td>\n"
			}

		} else if existeExtendida3 == true {

			file, err := os.OpenFile(nodoActual.ruta, os.O_RDONLY, 0644)
			if err != nil {
				fmt.Println("¡¡ Error !! No se pudo acceder al disco")
				respuesta = "¡¡ Error !! No se pudo acceder al disco"
				return respuesta
			}
			defer file.Close()

			if _, err := file.Seek(int64(nodoActual.inicioparticion), 0); err != nil {
				fmt.Println(err)
				return err.Error()
			}

			var PruebaEBR EBR
			if err := binary.Read(file, binary.LittleEndian, &PruebaEBR); err != nil {
				fmt.Println(err)
				return err.Error()
			}

			fsType2 := string(bytes.TrimRight(PruebaEBR.Part_name[:], string(0)))
			fmt.Println(fsType2)
			siguienteE := string(bytes.TrimRight(PruebaEBR.Part_next[:], string(0)))
			INT_P, err := strconv.Atoi(siguienteE)

			string_sizeP1 := string(bytes.TrimRight(particion[2].Part_size[:], string(0)))
			tamano_P1, err := strconv.Atoi(string_sizeP1)
			if err != nil {
				fmt.Println("Error:", err)
			}
			porcentajeP1 := (tamano_P1 * 100) / tamano
			if INT_P == -1 {
				dot = dot + "</tr>\n"
				dot = dot + "<tr>\n"
				nombrelog := string(bytes.TrimRight(PruebaEBR.Part_name[:], string(0)))
				size_p1 := string(bytes.TrimRight(PruebaEBR.Part_size[:], string(0)))
				int_size_p1, err := strconv.Atoi(size_p1)
				if err != nil {
					fmt.Println("Error:", err)
				}

				salidaP := (int_size_p1 * 100) / tamano
				porcentaje1 := strconv.Itoa(salidaP)

				espaciolibre := porcentajeP1 - salidaP
				str_espaciolinre := strconv.Itoa(espaciolibre)

				dot = dot + "<td rowspan=\"2\" port='log1'>EBR</td><td rowspan=\"2\" port='log2'>" + nombrelog + "<br/>" + porcentaje1 + "%" + "del disco</td>\n"
				dot = dot + "<td rowspan=\"2\" port='libre'>Espacio Libre <br/>" + str_espaciolinre + "%" + "del disco</td>\n"
			} else {
				dot = dot + "</tr>\n"
				dot = dot + "<tr>\n"
				nombrelog := string(bytes.TrimRight(PruebaEBR.Part_name[:], string(0)))
				size_p1 := string(bytes.TrimRight(PruebaEBR.Part_size[:], string(0)))
				int_size_p1, err := strconv.Atoi(size_p1)
				if err != nil {
					fmt.Println("Error:", err)
				}

				salidaP := (int_size_p1 * 100) / tamano
				porcentaje1 := strconv.Itoa(salidaP)

				if _, err := file.Seek(int64(nodoActual.inicioparticion+30), 0); err != nil {
					fmt.Println(err)
					return err.Error()
				}

				var SiguienteEBR EBR
				if err := binary.Read(file, binary.LittleEndian, &SiguienteEBR); err != nil {
					fmt.Println(err)
					return err.Error()
				}

				nombrelog2 := string(bytes.TrimRight(SiguienteEBR.Part_name[:], string(0)))
				size_p2 := string(bytes.TrimRight(SiguienteEBR.Part_size[:], string(0)))
				int_size_p2, err := strconv.Atoi(size_p2)
				if err != nil {
					fmt.Println("Error:", err)
				}

				salidaP2 := (int_size_p2 * 100) / tamano
				porcentaje2 := strconv.Itoa(salidaP2)

				espaciolibre := porcentajeP1 - (salidaP + salidaP2)
				str_espaciolinre := strconv.Itoa(espaciolibre)

				dot = dot + "<td rowspan=\"2\" port='log1'>EBR</td><td rowspan=\"2\" port='log'>" + nombrelog + "<br/>" + porcentaje1 + "%" + "del disco</td>\n"

				dot = dot + "<td rowspan=\"2\" port='log1'>EBR</td><td rowspan=\"2\" port='log2'>" + nombrelog2 + "<br/>" + porcentaje2 + "%" + "del disco</td>\n"

				dot = dot + "<td rowspan=\"2\" port='libre'>Espacio Libre <br/>" + str_espaciolinre + "%" + "del disco</td>\n"
			}

		} else if existeExtendida4 == true {

			file, err := os.OpenFile(nodoActual.ruta, os.O_RDONLY, 0644)
			if err != nil {
				fmt.Println("¡¡ Error !! No se pudo acceder al disco")
				respuesta = "¡¡ Error !! No se pudo acceder al disco"
				return respuesta
			}
			defer file.Close()

			if _, err := file.Seek(int64(nodoActual.inicioparticion), 0); err != nil {
				fmt.Println(err)
				return err.Error()
			}

			var PruebaEBR EBR
			if err := binary.Read(file, binary.LittleEndian, &PruebaEBR); err != nil {
				fmt.Println(err)
				return err.Error()
			}

			fsType2 := string(bytes.TrimRight(PruebaEBR.Part_name[:], string(0)))
			fmt.Println(fsType2)
			siguienteE := string(bytes.TrimRight(PruebaEBR.Part_next[:], string(0)))
			INT_P, err := strconv.Atoi(siguienteE)

			string_sizeP1 := string(bytes.TrimRight(particion[3].Part_size[:], string(0)))
			tamano_P1, err := strconv.Atoi(string_sizeP1)
			if err != nil {
				fmt.Println("Error:", err)
			}
			porcentajeP1 := (tamano_P1 * 100) / tamano
			if INT_P == -1 {
				dot = dot + "</tr>\n"
				dot = dot + "<tr>\n"
				nombrelog := string(bytes.TrimRight(PruebaEBR.Part_name[:], string(0)))
				size_p1 := string(bytes.TrimRight(PruebaEBR.Part_size[:], string(0)))
				int_size_p1, err := strconv.Atoi(size_p1)
				if err != nil {
					fmt.Println("Error:", err)
				}

				salidaP := (int_size_p1 * 100) / tamano
				porcentaje1 := strconv.Itoa(salidaP)

				espaciolibre := porcentajeP1 - salidaP
				str_espaciolinre := strconv.Itoa(espaciolibre)

				dot = dot + "<td rowspan=\"2\" port='log1'>EBR</td><td rowspan=\"2\" port='log2'>" + nombrelog + "<br/>" + porcentaje1 + "%" + "del disco</td>\n"
				dot = dot + "<td rowspan=\"2\" port='libre'>Espacio Libre <br/>" + str_espaciolinre + "%" + "del disco</td>\n"
			} else {
				dot = dot + "</tr>\n"
				dot = dot + "<tr>\n"
				nombrelog := string(bytes.TrimRight(PruebaEBR.Part_name[:], string(0)))
				size_p1 := string(bytes.TrimRight(PruebaEBR.Part_size[:], string(0)))
				int_size_p1, err := strconv.Atoi(size_p1)
				if err != nil {
					fmt.Println("Error:", err)
				}

				salidaP := (int_size_p1 * 100) / tamano
				porcentaje1 := strconv.Itoa(salidaP)

				if _, err := file.Seek(int64(nodoActual.inicioparticion+30), 0); err != nil {
					fmt.Println(err)
					return err.Error()
				}

				var SiguienteEBR EBR
				if err := binary.Read(file, binary.LittleEndian, &SiguienteEBR); err != nil {
					fmt.Println(err)
					return err.Error()
				}

				nombrelog2 := string(bytes.TrimRight(SiguienteEBR.Part_name[:], string(0)))
				size_p2 := string(bytes.TrimRight(SiguienteEBR.Part_size[:], string(0)))
				int_size_p2, err := strconv.Atoi(size_p2)
				if err != nil {
					fmt.Println("Error:", err)
				}

				salidaP2 := (int_size_p2 * 100) / tamano
				porcentaje2 := strconv.Itoa(salidaP2)

				espaciolibre := porcentajeP1 - (salidaP + salidaP2)
				str_espaciolinre := strconv.Itoa(espaciolibre)

				dot = dot + "<td rowspan=\"2\" port='log1'>EBR</td><td rowspan=\"2\" port='log'>" + nombrelog + "<br/>" + porcentaje1 + "%" + "del disco</td>\n"

				dot = dot + "<td rowspan=\"2\" port='log1'>EBR</td><td rowspan=\"2\" port='log2'>" + nombrelog2 + "<br/>" + porcentaje2 + "%" + "del disco</td>\n"

				dot = dot + "<td rowspan=\"2\" port='libre'>Espacio Libre <br/>" + str_espaciolinre + "%" + "del disco</td>\n"
			}

		}

	}

	dot = dot + "</tr>\n"
	dot = dot + "</table>\n"
	dot = dot + ">];\n"
	dot = dot + "}\n"

	archivo, err := os.Create("ReporteDisk.dot")
	if err != nil {
		fmt.Println(err)
	}
	defer archivo.Close()

	_, err = archivo.WriteString(dot)
	if err != nil {
		fmt.Println(err)
	}

	comando_ejecutar := "dot -Tpng ReporteDisk.dot -o " + directorio + nombreSinExt + ".png"

	cmd := exec.Command("/bin/sh", "-c", comando_ejecutar)

	err = cmd.Run()
	if err != nil {
		fmt.Println("Error al ejecutar el comando:", err)
		//return
	}

	fmt.Println("")
	fmt.Println("*               Reporte Disk creado con exito              *")
	fmt.Println("")

	ruta_buscar := directorio + nombreSinExt + ".png"

	bytes, _ := ioutil.ReadFile(ruta_buscar)
	var base64Encoding string
	base64Encoding += "data:image/png;base64,"
	base64Encoding += toBase64(bytes)
	respuesta = base64Encoding

	return respuesta
}

func reporte_tree(nodoActual *NodoMount, rutaa string) string {
	respuesta := ""

	// Se lee el contenido del primer archivo
	sakee, err := verSB(nodoActual.ruta, nodoActual.inicioparticion)
	if err != nil {
		// manejar error
	}

	lines := strings.Split(sakee, "\n")

	// for _, line := range lines {
	// 	fmt.Println(line)
	// }

	nombreedisco := filepath.Base(nodoActual.ruta)
	nombre := filepath.Base(rutaa)                                   // obtiene el nombre del archivo con la extensión
	nombreSinExt := strings.TrimSuffix(nombre, filepath.Ext(nombre)) // remueve la extensión

	// Creacion, escritura y cierre de archivo
	directorio := rutaa[:strings.LastIndex(rutaa, "/")+1]

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

	fmt.Println("Aqui va el codigo para el tree")

	dot := "digraph G {\n"
	dot = dot + "labelloc=\"t\"\n"
	dot = dot + "label=\"" + nombreedisco + "\"\n"
	dot = dot + "rankdir = \"LR\"\n"
	dot = dot + "\"node0\" [label=\"<f0>Inodo 0|<f1>i_type: 0|<f2>Ap0: Bloq0|<f3>Ap1: -1|<f3>Ap3: -1|<f4>...|<f5>Ap15: -1\" shape=\"record\" style=filled fillcolor=\"cadetblue1\"]\n"
	dot = dot + "\"node1\" [label=\"<f0>B. Carpeta 0|<f1>Users.txt: Ino1|<f2>home: Ino2|<f3>. : -1|<f4>. : -1\" shape=\"record\" style=filled fillcolor=\"darkolivegreen1\"]\n"
	dot = dot + "\"node2\" [label=\"<f0>Inodo 1|<f1>i_type: 1|<f2>Ap0: Bloq1|<f3>Ap1: -1|<f3>Ap3: -1|<f4>...|<f5>Ap15: -1\" shape=\"record\" style=filled fillcolor=\"cadetblue1\"]\n"
	dot = dot + "\"node3\" [label=\"<f0>B. Archivo 1|<f1>"
	for _, line := range lines {
		dot = dot + " " + line + "\n"
	}
	dot = dot + "\"shape=\"record\" style=filled fillcolor=\"gold\"];\n"
	dot = dot + "\"node0\":f2 -> \"node1\":f0;\n"
	dot = dot + "\"node1\":f1 -> \"node2\":f0;\n"
	dot = dot + "\"node2\":f2 -> \"node3\":f0;\n"
	dot = dot + "}\n"

	archivo, err := os.Create("ReporteTree.dot")
	if err != nil {
		fmt.Println(err)
		//return err.Error()
	}
	defer archivo.Close()

	_, err = archivo.WriteString(dot)
	if err != nil {
		fmt.Println(err)
		//return err.Error()
	}

	comando_ejecutar := "dot -Tpng ReporteTree.dot -o " + directorio + nombreSinExt + ".png"

	cmd := exec.Command("/bin/sh", "-c", comando_ejecutar)

	err = cmd.Run()
	if err != nil {
		fmt.Println("Error al ejecutar el comando:", err)
		//return err.Error()
	}

	fmt.Println("")
	fmt.Println("*              Reporte Tree creado con exito               *")
	fmt.Println("")

	ruta_buscar := directorio + nombreSinExt + ".png"

	bytes, _ := ioutil.ReadFile(ruta_buscar)
	var base64Encoding string
	base64Encoding += "data:image/png;base64,"
	base64Encoding += toBase64(bytes)
	respuesta = base64Encoding

	return respuesta
}

func reporte_sb(nodoActual *NodoMount, rutaa string) string {
	respuesta := ""
	Tam_EBR := EBR{}
	EBR_Size := unsafe.Sizeof(Tam_EBR)

	file, err := os.OpenFile(nodoActual.ruta, os.O_RDONLY, 0644)
	if err != nil {
		fmt.Println("¡¡ Error !! No se pudo acceder al disco")
		respuesta = "¡¡ Error !! No se pudo acceder al disco"
		return respuesta
	}
	defer file.Close()

	nuevoinicio := nodoActual.inicioparticion + int(EBR_Size)

	if _, err := file.Seek(int64(nuevoinicio), 0); err != nil {
		fmt.Println(err)
	}

	var SB SuperBloque
	if err := binary.Read(file, binary.LittleEndian, &SB); err != nil {
		fmt.Println(err)
	}

	sb_files := string(bytes.TrimRight(SB.S_filesystem_type[:], string(0)))
	sb_inodes := string(bytes.TrimRight(SB.S_inodes_count[:], string(0)))
	sb_block := string(bytes.TrimRight(SB.S_blocks_count[:], string(0)))
	sb_free_block := string(bytes.TrimRight(SB.S_free_blocks_count[:], string(0)))
	sb_free_inodes := string(bytes.TrimRight(SB.S_free_inodes_count[:], string(0)))
	sb_mtime := string(bytes.TrimRight(SB.S_mtime[:], string(0)))
	sb_mnt := string(bytes.TrimRight(SB.S_mnt_count[:], string(0)))
	sb_magic := string(bytes.TrimRight(SB.S_magic[:], string(0)))
	sb_inode_size := string(bytes.TrimRight(SB.S_inode_size[:], string(0)))
	sb_block_size := string(bytes.TrimRight(SB.S_block_size[:], string(0)))
	sb_first_ino := string(bytes.TrimRight(SB.S_firts_ino[:], string(0)))
	sb_first_blo := string(bytes.TrimRight(SB.S_first_blo[:], string(0)))
	sb_bm_inode := string(bytes.TrimRight(SB.S_bm_inode_start[:], string(0)))
	sb_bm_block := string(bytes.TrimRight(SB.S_bm_block_start[:], string(0)))
	sb_inode_start := string(bytes.TrimRight(SB.S_inode_start[:], string(0)))
	sb_block_start := string(bytes.TrimRight(SB.S_block_start[:], string(0)))

	nombreedisco := filepath.Base(nodoActual.ruta)
	nombre := filepath.Base(rutaa)                                   // obtiene el nombre del archivo con la extensión
	nombreSinExt := strings.TrimSuffix(nombre, filepath.Ext(nombre)) // remueve la extensión

	// Creacion, escritura y cierre de archivo
	directorio := rutaa[:strings.LastIndex(rutaa, "/")+1]

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

	dot := "digraph G {\n"
	dot = dot + "labelloc=\"t\"\n"
	dot = dot + "label=\"" + nombreedisco + "\"\n"
	dot = dot + "parent [\n"
	dot = dot + "shape=plaintext\n"
	dot = dot + "label=<\n"
	dot = dot + "<table border=\"1\" cellborder=\"1\">\n"
	dot = dot + "<tr><td bgcolor=\"darkgreen\" colspan=\"3\">REPORTE DE SUPERBLOQUE</td></tr>\n"
	dot = dot + "<tr><td port='fl'>s_filesystem_type</td><td port='siz1'>" + sb_files + "</td></tr>\n"
	dot = dot + "<tr><td bgcolor=\"forestgreen\" port=\"count\">s_inodes_count</td><td bgcolor=\"forestgreen\" port=\"siz17\">" + sb_inodes + "</td></tr>\n"
	dot = dot + "<tr><td port='bcount'>s_blocks_count</td><td port='siz2'>" + sb_block + "</td></tr>\n"
	dot = dot + "<tr><td bgcolor=\"forestgreen\" port=\"freeblocks\">s_free_blocks_count</td><td bgcolor=\"forestgreen\" port=\"siz16\">" + sb_free_block + "</td></tr>\n"
	dot = dot + "<tr><td port='freeinodes'>s_free_inodes_count</td><td port='siz3'>" + sb_free_inodes + "</td></tr>\n"
	dot = dot + "<tr><td bgcolor=\"forestgreen\" port=\"mounttime\">s_mtime</td><td bgcolor=\"forestgreen\" port=\"size15\">" + sb_mtime + "</td></tr>\n"
	dot = dot + "<tr><td bgcolor=\"forestgreen\" port='mountcount'>s_mnt_count</td><td bgcolor=\"forestgreen\" port='siz14'>" + sb_mnt + "</td></tr>\n"
	dot = dot + "<tr><td port='magic'>s_magic</td><td port='siz5'>" + sb_magic + "</td></tr>\n"
	dot = dot + "<tr><td bgcolor=\"forestgreen\" port=\"inodes\">s_inode_s</td><td bgcolor=\"forestgreen\" port=\"siz13\">" + sb_inode_size + "</td></tr>\n"
	dot = dot + "<tr><td port='sblock'>s_block_s</td><td port='siz6'>" + sb_block_size + "</td></tr>\n"
	dot = dot + "<tr><td bgcolor=\"forestgreen\" port=\"sfirstino\">s_firts_ino</td><td bgcolor=\"forestgreen\" port=\"siz12\">" + sb_first_ino + "</td></tr>\n"
	dot = dot + "<tr><td port='sfirstblo'>s_first_blo</td><td port='siz7'>" + sb_first_blo + "</td></tr>\n"
	dot = dot + "<tr><td bgcolor=\"forestgreen\" port='bminodes'>s_bm_inode_start</td><td bgcolor=\"forestgreen\" port='siz11'>" + sb_bm_inode + "</td></tr>\n"
	dot = dot + "<tr><td port='bmblocks'>s_bm_block_start</td><td port='siz8'>" + sb_bm_block + "</td></tr>\n"
	dot = dot + "<tr><td bgcolor=\"forestgreen\" port='inodestart'>s_inode_start</td><td bgcolor=\"forestgreen\" port='siz10'>" + sb_inode_start + "</td></tr>\n"
	dot = dot + "<tr><td port='blockstart'>s_block_start</td><td port='siz9'>" + sb_block_start + "</td></tr>\n"
	dot = dot + "</table>\n"
	dot = dot + ">];\n"
	dot = dot + "}\n"

	archivo, err := os.Create("ReporteSB.dot")
	if err != nil {
		fmt.Println(err)
		return err.Error()
	}
	defer archivo.Close()

	_, err = archivo.WriteString(dot)
	if err != nil {
		fmt.Println(err)
		return err.Error()
	}

	comando_ejecutar := "dot -Tpng ReporteSB.dot -o " + directorio + nombreSinExt + ".png"

	cmd := exec.Command("/bin/sh", "-c", comando_ejecutar)

	err = cmd.Run()
	if err != nil {
		fmt.Println("Error al ejecutar el comando:", err)
		return err.Error()
	}

	fmt.Println("")
	fmt.Println("*          Reporte SuperBloque creado con exito            *")
	fmt.Println("")

	ruta_buscar := directorio + nombreSinExt + ".png"

	bytes, _ := ioutil.ReadFile(ruta_buscar)
	var base64Encoding string
	base64Encoding += "data:image/png;base64,"
	base64Encoding += toBase64(bytes)
	respuesta = base64Encoding

	return respuesta
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

func cerrar_sesion() {
	for i := 0; i < len(usuarioLogeado); i++ {
		usuarioLogeado[i] = NodoLogin{}
	}
	usuarioLogeado = usuarioLogeado[:0]
}

func findMax(nums []int) int {
	max := nums[0]
	for _, n := range nums {
		if n > max {
			max = n
		}
	}
	return max
}
