package main

import (
	"bufio"
	"bytes"
	"encoding/gob"
	"fmt"
	"io"
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
	Part_status [100]byte
	Part_type   [100]byte
	Part_fit    [100]byte
	Part_start  [100]byte
	Part_size   [100]byte
	Part_name   [100]byte
}

type MBR = struct {
	Mbr_tamano         [100]byte
	Mbr_fecha_creacion [100]byte
	Mbr_dsk_signature  [100]byte
	Dsk_fit            [100]byte
	Mbr_partition_1    Partition
	Mbr_partition_2    Partition
	Mbr_partition_3    Partition
	Mbr_partition_4    Partition
}

type EBR = struct {
	Part_status [100]byte
	Part_fit    [100]byte
	Part_start  [100]byte
	Part_size   [100]byte
	Part_next   [100]byte
	Part_name   [100]byte
}

// Esto ayuda para el montaje de las particiones

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
	} else {
		fmt.Println("Comando ingresado no es valido")
	}
	//else if data == "escribir" {
	//	escribir(commandArray)
	//} else if data == "mostrar" {
	//	mostrar()
	//} else if data == "registrox" {
	//	registrox(commandArray)
	//}

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

	//fmt.Println("Numero = ", dsk_s)

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

	// Conversion de struct a bytes
	ejmbyte := struct_to_bytes(Disco)
	// Cambio de posicion de puntero dentro del archivo
	newpos, err := disco.Seek(0, os.SEEK_SET)
	if err != nil {
		msg_error(err)
	}
	// Escritura de struct en archivo binario
	_, err = disco.WriteAt(ejmbyte, newpos)
	if err != nil {
		msg_error(err)
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
			//ruta = data
			//fmt.Println("Ahora? ", ruta)
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
		//fmt.Printf("¡¡ Error !! al buscar y eliminar el archivo: %v\n", err)
		fmt.Println("¡¡ Error !! No se ha encontrado el archivo", err)
	}

}

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

	//fmt.Println("Datos ")
	//fmt.Println(tamano_part)
	//fmt.Println(reflect.TypeOf(tamano_part))
	//fmt.Println(part_unit)
	//fmt.Println(rutaa)
	//fmt.Println(part_type)
	//fmt.Println(part_fit)
	//fmt.Println(nombre_part)

	fin_archivo := false
	var emptymbr [100]byte
	ejm_empty := MBR{}
	// Apertura de archivo
	disco, err := os.OpenFile(rutaa, os.O_RDWR, 0660)
	if err != nil {
		msg_error(err)
	}
	// Calculo del tamano de struct en bytes

	string_tamano := ""
	//string_fecha := ""
	//string_dsk := ""
	//string_ajuste := ""
	var solaa MBR

	mbr2 := struct_to_bytes(ejm_empty)
	sstruct := len(mbr2)
	if !fin_archivo {
		// Lectrura de conjunto de bytes en archivo binario
		lectura := make([]byte, sstruct)
		_, err = disco.ReadAt(lectura, 0)
		if err != nil && err != io.EOF {
			msg_error(err)
		}
		// Conversion de bytes a struct
		mbr := bytes_to_struct_mbr(lectura)

		solaa = mbr
		sstruct = len(lectura)
		if err != nil {
			msg_error(err)
		}
		if mbr.Mbr_tamano == emptymbr {
			fin_archivo = true
		} else {
			//fmt.Println("Sacando los datos del .dsk")
			//fmt.Print(" Tamaño: ")
			//fmt.Print(string(ejm.Mbr_tamano[:]))
			string_tamano = string(mbr.Mbr_tamano[:])
			//fmt.Print(" Fecha: ")
			//fmt.Print(string(ejm.Mbr_fecha_creacion[:]))
			//string_fecha = string(mbr.Mbr_fecha_creacion[:])
			//fmt.Print(" DSK: ")
			//fmt.Print(string(ejm.Mbr_dsk_signature[:]))
			//string_dsk = string(mbr.Mbr_dsk_signature[:])
			//fmt.Print(" Ajuste: ")
			//fmt.Println(string(ejm.Dsk_fit[:]))
			//string_ajuste = string(mbr.Dsk_fit[:])
		}
	}
	disco.Close()

	//fmt.Println(string_tamano)
	//fmt.Println(reflect.TypeOf(string_tamano))
	//fmt.Println(string_fecha)
	//fmt.Println(string_dsk)
	//fmt.Println(string_ajuste)

	trimmed_string_tamano := strings.TrimRightFunc(string_tamano, func(r rune) bool { return r == '\x00' })
	tamano, err := strconv.Atoi(trimmed_string_tamano)
	if err != nil {
		fmt.Println("Error:", err)
	}

	//fmt.Println(reflect.TypeOf(tamano))
	discop := solaa

	particion := [4]Partition{
		discop.Mbr_partition_1,
		discop.Mbr_partition_2,
		discop.Mbr_partition_3,
		discop.Mbr_partition_4,
	}

	if tamano >= tamano_archivo1 {
		//fmt.Println(tamano)
		//fmt.Println(tamano_archivo1)
		//fmt.Println("Hay espacio suficiente")

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
						discop.Mbr_partition_1.Part_status = [100]byte{'1'}
						//fmt.Println("Entra al 1")
					case 1:
						discop.Mbr_partition_2.Part_status = [100]byte{'1'}
						//fmt.Println("Entra al 2")
					case 2:
						discop.Mbr_partition_3.Part_status = [100]byte{'1'}
						//fmt.Println("Entra al 3")
					case 3:
						discop.Mbr_partition_4.Part_status = [100]byte{'1'}
						//fmt.Println("Entra al 4")
					}
				}
			}

			fmt.Println("Aver ", string(discop.Mbr_partition_1.Part_status[:]))
			fmt.Println("Aver2 ", string(discop.Mbr_partition_2.Part_status[:]))
			fmt.Println("Aver3 ", string(discop.Mbr_partition_3.Part_status[:]))
			fmt.Println("Aver4 ", string(discop.Mbr_partition_4.Part_status[:]))

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
					discoescritura, err := os.OpenFile(rutaa, os.O_RDWR, 0660)
					if err != nil {
						msg_error(err)
					}

					// Conversion de struct a bytes
					ejmbyte := struct_to_bytes(discop)
					// Cambio de posicion de puntero dentro del archivo
					newpos, err := discoescritura.Seek(0, os.SEEK_SET)
					if err != nil {
						msg_error(err)
					}
					// Escritura de struct en archivo binario
					_, err = discoescritura.WriteAt(ejmbyte, newpos)
					if err != nil {
						msg_error(err)
					}

					discoescritura.Close()
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
				if err != nil {
					// Manejo del error
				}
				tamanoDisponibleAntes1 := tamano - partSizeInt
				fmt.Println("Tamano disponible? ", tamanoDisponibleAntes1)
				fmt.Println("Tamano a asignar ", tamano_archivo1)
				resta := tamanoDisponibleAntes1 - tamano_archivo1
				fmt.Println("Resta ", resta)
				if tamanoDisponibleAntes1 >= tamano_archivo1 {
					//fmt.Println("Si hay espacio para la particion")
					copy(discop.Mbr_partition_1.Part_status[:], "0")
					copy(discop.Mbr_partition_2.Part_status[:], "0")
					copy(discop.Mbr_partition_2.Part_type[:], part_type)
					copy(discop.Mbr_partition_2.Part_fit[:], part_fit)
					unaprueba := string(particion[0].Part_start[:])
					unaprueba = strings.TrimRightFunc(unaprueba, func(r rune) bool { return r == '\x00' })
					intprueba, err := strconv.Atoi(unaprueba)
					startt := intprueba + partSizeInt + 1
					fmt.Println("Inicio? ", intprueba)
					fmt.Println("Size? ", partSizeInt)
					fmt.Println("Start? ", startt)
					copy(discop.Mbr_partition_2.Part_start[:], strconv.Itoa(startt))
					//disco.mbr_partition_2.part_start = (disco.mbr_partition_1.part_start + disco.mbr_partition_1.part_s + 1);
					copy(discop.Mbr_partition_2.Part_size[:], strconv.Itoa(tamano_archivo1))
					copy(discop.Mbr_partition_2.Part_name[:], nombre_part)

					//Apertura del archivo
					discoescritura, err := os.OpenFile(rutaa, os.O_RDWR, 0660)
					if err != nil {
						msg_error(err)
					}

					// Conversion de struct a bytes
					ejmbyte2 := struct_to_bytes(discop)
					// Cambio de posicion de puntero dentro del archivo
					newpos2, err := discoescritura.Seek(0, os.SEEK_SET)
					if err != nil {
						msg_error(err)
					}
					// Escritura de struct en archivo binario
					_, err = discoescritura.WriteAt(ejmbyte2, newpos2)
					if err != nil {
						msg_error(err)
					}

					discoescritura.Close()
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
				if err != nil {
					// Manejo del error
				}
				tamanoDisponibleAntes1 := tamano - (partSizeInt + partSizeInt1)
				fmt.Println("Tamano disponible? ", tamanoDisponibleAntes1)
				fmt.Println("Tamano a asignar ", tamano_archivo1)
				resta := tamanoDisponibleAntes1 - tamano_archivo1
				fmt.Println("Resta ", resta)
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
					startt := intprueba + partSizeInt1 + 1
					fmt.Println("Inicio? ", intprueba)
					fmt.Println("Size? ", partSizeInt)
					fmt.Println("Start? ", startt)
					copy(discop.Mbr_partition_3.Part_start[:], strconv.Itoa(startt))
					copy(discop.Mbr_partition_3.Part_size[:], strconv.Itoa(tamano_archivo1))
					copy(discop.Mbr_partition_3.Part_name[:], nombre_part)

					//Apertura del archivo
					discoescritura, err := os.OpenFile(rutaa, os.O_RDWR, 0660)
					if err != nil {
						msg_error(err)
					}

					// Conversion de struct a bytes
					ejmbyte3 := struct_to_bytes(discop)
					// Cambio de posicion de puntero dentro del archivo
					newpos3, err := discoescritura.Seek(0, os.SEEK_SET)
					if err != nil {
						msg_error(err)
					}
					// Escritura de struct en archivo binario
					_, err = discoescritura.WriteAt(ejmbyte3, newpos3)
					if err != nil {
						msg_error(err)
					}

					discoescritura.Close()
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
				if err != nil {
					// Manejo del error
				}
				tamanoDisponibleAntes1 := tamano - (partSizeInt + partSizeInt1 + partSizeInt2)
				fmt.Println("Tamano disponible? ", tamanoDisponibleAntes1)
				fmt.Println("Tamano a asignar ", tamano_archivo1)
				resta := tamanoDisponibleAntes1 - tamano_archivo1
				fmt.Println("Resta ", resta)
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
					startt := intprueba + partSizeInt2 + 1
					fmt.Println("Inicio? ", intprueba)
					fmt.Println("Size? ", partSizeInt)
					fmt.Println("Start? ", startt)
					copy(discop.Mbr_partition_4.Part_start[:], strconv.Itoa(startt))
					copy(discop.Mbr_partition_4.Part_size[:], strconv.Itoa(tamano_archivo1))
					copy(discop.Mbr_partition_4.Part_name[:], nombre_part)

					//Apertura del archivo
					discoescritura, err := os.OpenFile(rutaa, os.O_RDWR, 0660)
					if err != nil {
						msg_error(err)
					}

					// Conversion de struct a bytes
					ejmbyte4 := struct_to_bytes(discop)
					// Cambio de posicion de puntero dentro del archivo
					newpos4, err := discoescritura.Seek(0, os.SEEK_SET)
					if err != nil {
						msg_error(err)
					}
					// Escritura de struct en archivo binario
					_, err = discoescritura.WriteAt(ejmbyte4, newpos4)
					if err != nil {
						msg_error(err)
					}

					discoescritura.Close()
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
			//Empiezan las particiones logicas
			Prueba_EBR := EBR{}
			var obtener_ebr EBR
			encontrado := false
			encontrado1 := false
			encontrado2 := false
			encontrado3 := false
			for i := 0; i < 4; i++ {
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
			}

			if encontrado == true {
				fmt.Println("Llega ?")
				//var emptyid [100]byte
				inicio_particion1 := string(particion[0].Part_start[:])
				inicio_particion1 = strings.TrimRightFunc(inicio_particion1, func(r rune) bool { return r == '\x00' })
				//int_inicio_particion1, err := strconv.Atoi(inicio_particion1)
				int_inicio_particion1, err := strconv.Atoi(inicio_particion1)

				// Apertura de archivo
				disco, err := os.OpenFile(rutaa, os.O_RDWR, 0660)
				if err != nil {
					msg_error(err)
				}
				// Calculo del tamano de struct en bytes
				ejm2 := struct_to_bytes(Prueba_EBR)
				sstruct := len(ejm2)
				for !fin_archivo {
					// Lectrura de conjunto de bytes en archivo binario
					lectura := make([]byte, sstruct)
					_, err = disco.ReadAt(lectura, int64(int_inicio_particion1))
					if err != nil && err != io.EOF {
						msg_error(err)
					}
					// Conversion de bytes a struct
					ejm := bytes_to_struct_ebr(lectura)
					obtener_ebr = ejm
					sstruct = len(lectura)
					if err != nil {
						msg_error(err)
					}
					if len(strings.TrimSpace(string(ejm.Part_fit[:]))) == 0 {
						fin_archivo = true
					} else {
						fmt.Print(" Status: ")
						fmt.Print(string(ejm.Part_status[:]))

						verificarlogicas := string(ejm.Part_name[:])
						verificarlogicas = strings.TrimRightFunc(verificarlogicas, func(r rune) bool { return r == '\x00' })

						if verificarlogicas == "" {
							copy(obtener_ebr.Part_status[:], "0")
							copy(obtener_ebr.Part_fit[:], part_fit)
							copy(obtener_ebr.Part_start[:], strconv.Itoa(int_inicio_particion1))
							copy(obtener_ebr.Part_size[:], strconv.Itoa(tamano_archivo1))
							copy(obtener_ebr.Part_next[:], strconv.Itoa(-1))
							copy(obtener_ebr.Part_name[:], nombre_part)

							fmt.Println("Aun no existe una particion logica")
							fmt.Println("Nombre a poner ", nombre_part)
							fmt.Println("Tamano ", tamano_archivo1)

							// Apertura del archivo
							discoescritura, err := os.OpenFile(rutaa, os.O_RDWR, 0660)
							if err != nil {
								msg_error(err)
							}

							// Conversion de struct a bytes
							ejmbyte := struct_to_bytes(obtener_ebr)
							// Cambio de posicion de puntero dentro del archivo
							newpos, err := discoescritura.Seek(int64(int_inicio_particion1), os.SEEK_SET)
							if err != nil {
								msg_error(err)
							}
							// Escritura de struct en archivo binario
							_, err = discoescritura.WriteAt(ejmbyte, newpos)
							if err != nil {
								msg_error(err)
							}

							discoescritura.Close()

							fmt.Println("Se ha guardado")
							return

						} else {
							fmt.Println("Entra cuando ya hay alguna particion")
							fmt.Println("Nombre a poner ", nombre_part)
							fmt.Println("Tamano ", tamano_archivo1)
							return
						}
					}

				}
				disco.Close()

				//fmt.Println("Entra al if encontrado 1")

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
	// tamano_parti := ""
	// straux := ""
	// unidad := ""
	rutaa := ""
	//ultimosdigitos := "16"
	//numeroparticion := ""

	// tipo_part := ""
	// ajuste_part := ""
	nombre_part := ""
	// tamano_part := 0
	// tamano_archivo1 := 0

	for i := 0; i < len(commandArray); i++ {
		data := commandArray[i]
		// if strings.HasPrefix(data, ">") {
		// 	// Convertir a minúsculas
		// 	data = strings.ToLower(data)
		// }

		if strings.Contains(data, ">path=") {
			rutaa = strings.Replace(data, ">path=", "", 1)
			//ruta = data
			//fmt.Println("Ahora? ", ruta)
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

	fmt.Println("Llega al comando mount")

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
